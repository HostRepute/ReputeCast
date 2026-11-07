package liquid

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/osteele/liquid/filters"
	"github.com/osteele/liquid/parser"
	"github.com/osteele/liquid/render"
	"github.com/osteele/liquid/tags"
	"github.com/osteele/liquid/values"
)

// Every render runs on a budget. Anyone who can preview an email can make
// the server render Liquid they wrote, so a template must not be able to
// keep a core busy or fill the memory: a loop over a range read from a
// variable, three nested loops of 1,000, or a string that doubles with
// append or capture on every pass were all once enough to stall or kill
// the process. The budget counts what a render does and stops it with an
// error when it does too much:
//
//   - steps: every tag, every output and every pass of a loop;
//   - work: the bytes filters and captures read and make;
//   - the size of one value (MaxValue) and of one list (MaxItems);
//   - time, for what the counts don't see (contains on a long string).
//
// The limits are far above what an email needs: a long digest takes a few
// thousand steps and well under a megabyte of work.
const (
	// MaxValue is the longest string a filter or capture may take or make.
	MaxValue = MaxOutput
	// MaxItems is the longest list or range a filter may take or make.
	MaxItems = 10_000
	// MaxWork is how many bytes the filters and captures of one render may
	// read and make in total.
	MaxWork = 64 << 20
	// MaxSteps is how many tags, outputs and loop passes one render may run.
	MaxSteps = 200_000
)

// MaxRenderTime is the longest one render may take.
var MaxRenderTime = 2 * time.Second

// ErrTooMuchWork is returned when a render goes over its budget.
var ErrTooMuchWork = errors.New("the personalization does too much work to render: loops run too often, or values grow too large")

// errNoBudget is returned when a template overwrote the budget variable.
var errNoBudget = errors.New("the personalization can't be rendered")

// budgetVar names the binding that carries a render's budget to its tags.
// Templates can't read anything from it; one that overwrites it only fails.
const budgetVar = "__rc_render"

// itemCost is what one list item counts for in the work budget.
const itemCost = 16

type budget struct {
	steps    int
	work     int
	deadline time.Time
	err      error
}

func newBudget() *budget { return &budget{deadline: time.Now().Add(MaxRenderTime)} }

// fail records the first reason the render stops and returns it.
func (b *budget) fail(err error) error {
	if b.err == nil {
		b.err = err
	}
	return b.err
}

// step counts a tag, an output or a loop pass.
func (b *budget) step() error {
	if b.err != nil {
		return b.err
	}
	b.steps++
	if b.steps > MaxSteps {
		return b.fail(ErrTooMuchWork)
	}
	// The clock is read every 32 steps: one step costs at most a
	// millisecond or so (contains on a 4 MB string), and reading it on
	// every step was most of what the budget cost a render.
	if b.steps%32 == 0 && time.Now().After(b.deadline) {
		return b.fail(ErrTooMuchWork)
	}
	return nil
}

// stopped reports whether the render has failed and stops at its next step.
func (b *budget) stopped() bool { return b.err != nil }

// spend counts n bytes of work and reports whether the budget allows it.
func (b *budget) spend(n int) bool { return b.charge(n) == nil }

// charge counts n bytes of work.
func (b *budget) charge(n int) error {
	if b.err != nil {
		return b.err
	}
	b.work += n
	if b.work > MaxWork {
		return b.fail(ErrTooMuchWork)
	}
	return nil
}

// value checks one value a filter takes or makes and counts it.
func (b *budget) value(v any) error {
	n, items := measure(v)
	if n > MaxValue || items > MaxItems {
		return b.fail(ErrTooMuchWork)
	}
	return b.charge(n + items*itemCost)
}

// measure returns the bytes of a string and the length of a list, map or
// range, without copying them. A range is measured before a filter turns
// it into a list: (1..50000000) | join would build fifty million strings.
func measure(v any) (bytes, items int) {
	switch v := v.(type) {
	case nil:
		return 0, 0
	case string:
		return len(v), 0
	case []byte:
		return len(v), 0
	case values.Range:
		return 0, v.Len()
	case values.SafeValue:
		return measure(v.Value)
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return 0, rv.Len()
	case reflect.String:
		return rv.Len(), 0
	}
	return 0, 0
}

// renderer is a Liquid configuration whose filters charge one render's
// budget. A render takes one from the pool, so each filter knows whose
// budget it spends without a lock.
type renderer struct {
	cfg render.Config
	b   *budget
}

var renderers = sync.Pool{New: func() any { return newRenderer() }}

// parseCfg parses every template (its filters are never called).
var parseCfg = newRenderer().cfg

func newRenderer() *renderer {
	r := &renderer{cfg: render.NewConfig()}
	filters.AddStandardFilters(chargedFilters{r})
	chargedFilters{r}.AddFilter("with_query", WithQuery)
	tags.AddStandardTags(&r.cfg)
	// osteele/liquid's include and render load other templates from the
	// template store, which by default reads files under the working
	// directory: an email could print the server's files, and a template
	// that renders itself recursed until the stack overflowed, killing the
	// process. Both tags are removed, so using one is a parse error with
	// its line, and the store holds no templates, so a tag added later
	// can't read files either.
	r.cfg.UnregisterTag("include")
	r.cfg.UnregisterTag("render")
	r.cfg.TemplateStore = noTemplates{}
	r.cfg.AddTag(tickTag, compileTick)
	r.cfg.AddBlock(captureTag).Compiler(compileCapture)
	return r
}

// chargedFilters registers filters that check and charge what they take
// and make.
type chargedFilters struct{ r *renderer }

func (c chargedFilters) AddFilter(name string, fn any) {
	f := reflect.ValueOf(fn)
	r := c.r
	r.cfg.AddFilter(name, func(in any, args ...any) (any, error) {
		b := r.b
		if b == nil {
			return nil, errNoBudget
		}
		if err := b.step(); err != nil {
			return nil, err
		}
		if err := b.value(in); err != nil {
			return nil, err
		}
		for _, a := range args {
			if err := b.value(a); err != nil {
				return nil, err
			}
		}
		out, err := values.Call(f, append([]any{in}, args...))
		if err != nil {
			return nil, err
		}
		if err := b.value(out); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// tickTag counts a step; Parse puts one before every tag and output, and
// at the top of every loop's body.
const tickTag = "rc_tick"

func compileTick(string) (func(io.Writer, render.Context) error, error) {
	return func(_ io.Writer, ctx render.Context) error {
		b, ok := ctx.Get(budgetVar).(*budget)
		if !ok {
			return errNoBudget
		}
		return b.step()
	}, nil
}

// captureTag is capture with a bounded buffer: osteele/liquid's capture
// renders into a buffer with no limit, so {{ s }}{{ s }} captured into s
// on every pass of a loop doubled the memory each time. Parse puts it in
// the place of every capture.
const captureTag = "rc_capture"

func compileCapture(node render.BlockNode) (func(io.Writer, render.Context) error, error) {
	name := strings.TrimSpace(node.Args)
	if name == "" || strings.ContainsAny(name, " \t") {
		return nil, fmt.Errorf("syntax error: capture tag requires exactly one variable name, got %q", node.Args)
	}
	return func(_ io.Writer, ctx render.Context) error {
		b, ok := ctx.Get(budgetVar).(*budget)
		if !ok {
			return errNoBudget
		}
		w := &captureWriter{b: b}
		if err := ctx.RenderChildren(w); err != nil {
			return err
		}
		if b.err != nil {
			return b.err
		}
		ctx.Set(name, w.s.String())
		return nil
	}, nil
}

// captureWriter keeps what a capture renders. Like limitWriter it drops
// what goes over the budget instead of failing the write (tablerow panics
// on a failed write); the capture then fails at its end.
type captureWriter struct {
	b *budget
	s strings.Builder
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if w.b.stopped() {
		return len(p), nil
	}
	if w.s.Len()+len(p) > MaxValue {
		_ = w.b.fail(ErrTooMuchWork)
		return len(p), nil
	}
	if !w.b.spend(len(p)) {
		return len(p), nil
	}
	return w.s.Write(p)
}

var (
	tickOnce sync.Once
	tickNode render.Node
)

// tick returns the node that counts a step (one, shared by every template:
// it keeps no state of its own).
func tick() render.Node {
	tickOnce.Do(func() {
		root, err := parseCfg.Compile("{% "+tickTag+" %}", parser.SourceLoc{})
		if err != nil {
			panic(err)
		}
		tickNode = firstNode(root, func(n render.Node) bool { _, ok := n.(*render.TagNode); return ok })
		if tickNode == nil {
			panic("liquid: no tick node")
		}
	})
	return tickNode
}

// boundedCapture is the rc_capture node that replaces a capture node: the
// same variable and the same body.
func boundedCapture(n *render.BlockNode) (*render.BlockNode, error) {
	root, err := parseCfg.Compile("{% "+captureTag+" "+n.Args+" %}{% end"+captureTag+" %}", n.SourceLoc)
	if err != nil {
		return nil, err
	}
	b, ok := firstNode(root, func(c render.Node) bool {
		b, ok := c.(*render.BlockNode)
		return ok && b.Name == captureTag
	}).(*render.BlockNode)
	if !ok {
		return nil, errors.New("liquid: no capture node")
	}
	b.Body = n.Body
	return b, nil
}

func firstNode(n render.Node, match func(render.Node) bool) render.Node {
	if match(n) {
		return n
	}
	switch n := n.(type) {
	case *render.SeqNode:
		for _, c := range n.Children {
			if m := firstNode(c, match); m != nil {
				return m
			}
		}
	case *render.BlockNode:
		for _, c := range n.Body {
			if m := firstNode(c, match); m != nil {
				return m
			}
		}
	}
	return nil
}

// instrument puts a tick before every tag, output and block of a parsed
// template and at the top of every loop's body, and swaps each capture for
// rc_capture.
//
// Nodes are replaced in place, never inserted, except in loop bodies: an
// if tag keeps a copy of its first branch's slice from when it was
// compiled, so only changes to the elements show. A loop reads its body
// from the node each time it renders.
func instrument(nodes []render.Node) error {
	for i, n := range nodes {
		switch n := n.(type) {
		case *render.ObjectNode, *render.TagNode:
			nodes[i] = ticked(n)
		case *render.SeqNode:
			if err := instrument(n.Children); err != nil {
				return err
			}
		case *render.BlockNode:
			b := n
			if n.Name == "capture" {
				var err error
				if b, err = boundedCapture(n); err != nil {
					return err
				}
			}
			if err := instrument(b.Body); err != nil {
				return err
			}
			for _, c := range b.Clauses {
				if err := instrument(c.Body); err != nil {
					return err
				}
			}
			if b.Name == "for" || b.Name == "tablerow" {
				b.Body = append([]render.Node{tick()}, b.Body...)
			}
			nodes[i] = ticked(b) //nolint:gosec // G602: i comes from ranging over nodes
		}
	}
	return nil
}

func ticked(n render.Node) render.Node {
	return &render.SeqNode{Children: []render.Node{tick(), n}}
}
