package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
)

// Catalog is one package's messages: the English source (messages/en.json)
// and its translations (messages/vi.json…).
type Catalog struct {
	Namespace string
	sources   map[string]*compiled
	trans     map[Locale]map[string]*compiled
}

type compiled struct {
	raw   string
	nodes []node
}

// entry is a message in a catalog file: "text", or {"message": "text", "note": "…"}.
type entry struct {
	Message string `json:"message"`
	Note    string `json:"note,omitempty"`
}

func (e *entry) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		e.Message = s
		return nil
	}
	type plain entry
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*e = entry(p)
	return nil
}

var (
	registryMu sync.Mutex
	registry   = map[string]*Catalog{}
)

// Catalogs returns every catalog loaded so far, by namespace.
func Catalogs() []*Catalog {
	registryMu.Lock()
	defer registryMu.Unlock()
	out := make([]*Catalog, 0, len(registry))
	for _, c := range registry {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Namespace < out[j].Namespace })
	return out
}

// Load reads messages/*.json from fsys. en.json is required. A translation
// may only use keys and arguments its English source has.
func Load(fsys fs.FS, namespace string) (*Catalog, error) {
	c := &Catalog{Namespace: namespace, sources: map[string]*compiled{}, trans: map[Locale]map[string]*compiled{}}
	src, err := readCatalog(fsys, "messages/en.json")
	if err != nil {
		return nil, fmt.Errorf("i18n %s: %w", namespace, err)
	}
	for k, e := range src {
		nodes, err := parse(e.Message)
		if err != nil {
			return nil, fmt.Errorf("i18n %s.%s: %w", namespace, k, err)
		}
		c.sources[k] = &compiled{raw: e.Message, nodes: nodes}
	}
	files, _ := fs.Glob(fsys, "messages/*.json")
	for _, f := range files {
		code := strings.TrimSuffix(path.Base(f), ".json")
		if code == "en" {
			continue
		}
		l := Locale(code)
		if !isAppLocale(l) || l == English {
			return nil, fmt.Errorf("i18n %s: %s isn't an app language", namespace, f)
		}
		tr, err := readCatalog(fsys, f)
		if err != nil {
			return nil, fmt.Errorf("i18n %s: %w", namespace, err)
		}
		c.trans[l] = map[string]*compiled{}
		for k, e := range tr {
			s, ok := c.sources[k]
			if !ok {
				return nil, fmt.Errorf("i18n %s: %s has %q, which en.json doesn't", namespace, f, k)
			}
			nodes, err := parse(e.Message)
			if err != nil {
				return nil, fmt.Errorf("i18n %s.%s (%s): %w", namespace, k, code, err)
			}
			want, got := map[string]bool{}, map[string]bool{}
			argNames(s.nodes, want)
			argNames(nodes, got)
			for name := range got {
				if !want[name] {
					return nil, fmt.Errorf("i18n %s.%s (%s): uses %s, which the English doesn't", namespace, k, code, name)
				}
			}
			c.trans[l][k] = &compiled{raw: e.Message, nodes: nodes}
		}
	}
	registryMu.Lock()
	registry[namespace] = c
	registryMu.Unlock()
	return c, nil
}

func readCatalog(fsys fs.FS, name string) (map[string]entry, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	out := map[string]entry{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// MustLoad is Load for package initialization: a broken catalog stops the
// program (and every test) at once.
func MustLoad(fsys fs.FS, namespace string) *Catalog {
	c, err := Load(fsys, namespace)
	if err != nil {
		panic(err)
	}
	return c
}

// M returns the message with this key. It panics when en.json doesn't have
// it, so a typo fails at package initialization.
func (c *Catalog) M(key string) Msg {
	if _, ok := c.sources[key]; !ok {
		panic(fmt.Sprintf("i18n: %s.%s isn't in %s's messages/en.json", c.Namespace, key, c.Namespace))
	}
	return Msg{c: c, key: key}
}

// Keys returns the message keys, sorted.
func (c *Catalog) Keys() []string {
	out := make([]string, 0, len(c.sources))
	for k := range c.sources {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Missing returns the keys a language hasn't translated yet.
func (c *Catalog) Missing(l Locale) []string {
	var out []string
	for _, k := range c.Keys() {
		if _, ok := c.trans[l][k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

// Msg is one message of a catalog.
type Msg struct {
	c   *Catalog
	key string
}

// ID is the message's full ID, namespace.key.
func (m Msg) ID() string { return m.c.Namespace + "." + m.key }

// Source is the English text.
func (m Msg) Source() string { return m.c.sources[m.key].raw }

// resolve picks the text for a language: its translation, else English.
func (m Msg) resolve(l Locale) (*compiled, Locale) {
	if l != English && l != Pseudo {
		if t, ok := m.c.trans[l][m.key]; ok {
			return t, l
		}
	}
	return m.c.sources[m.key], English
}

// Text formats the message as plain text: tags keep their content.
func (m Msg) Text(l Lang, args Args) string {
	return m.format(l, args, nil, modeText)
}

// HTML formats the message as HTML: its text and string values are
// escaped, HTML values are kept, and tags go through tags.
func (m Msg) HTML(l Lang, args Args, tags Tags) string {
	return m.format(l, args, tags, modeHTML)
}

func (m Msg) format(l Lang, args Args, tags Tags, md mode) string {
	l = l.norm()
	c, text := m.resolve(l.Locale)
	pseudo := l.Locale == Pseudo
	if pseudo {
		l.Region = Standard
	}
	f := &formatter{lang: l, text: text, args: args, tags: tags, mode: md, pseudo: pseudo}
	out := f.nodes(c.nodes, nil)
	if !pseudo {
		return out
	}
	plain := out
	if md == modeHTML {
		pf := *f
		pf.mode = modeText
		plain = pf.nodes(c.nodes, nil)
	}
	open, close := pseudoWrap(plain)
	return open + out + close
}

// Validate parses a message on its own (for tests and tools).
func Validate(message string) error {
	_, err := parse(message)
	return err
}

// Format formats a message that isn't in a catalog (for tests and tools).
func Format(l Lang, message string, args Args) (string, error) {
	nodes, err := parse(message)
	if err != nil {
		return "", err
	}
	l = l.norm()
	f := &formatter{lang: l, text: l.Locale, args: args, mode: modeText}
	return f.nodes(nodes, nil), nil
}

// ErrUnknownKey is returned by Lookup for a key the catalog doesn't have.
var ErrUnknownKey = errors.New("i18n: unknown message key")

// Lookup returns the message with this key, for keys only known at run time.
func (c *Catalog) Lookup(key string) (Msg, error) {
	if _, ok := c.sources[key]; !ok {
		return Msg{}, ErrUnknownKey
	}
	return Msg{c: c, key: key}, nil
}
