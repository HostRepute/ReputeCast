package products_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

// flows runs the automations engine on the shop's database, at the shop's
// time (orders are paid at it).
func (s *shop) flows() *automations.Service {
	a := s.env.App
	return &automations.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, Now: s.clock.now}
}

// publishFlow creates a flow with graph g through the API and publishes it.
func (s *shop) publishFlow(name string, g automations.AutomationGraph) uuid.UUID {
	s.t.Helper()
	var created struct{ ID string }
	s.owner.Do("POST", ws+"/automations", map[string]any{"name": name}).MustStatus(s.t, 201).JSON(s.t, &created)
	var saved struct {
		DraftVersion int
		Issues       []struct{ Code, Message string }
	}
	s.owner.Do("PUT", ws+"/automations/"+created.ID+"/draft", map[string]any{"graph": g, "revision": 0}).MustStatus(s.t, 200).JSON(s.t, &saved)
	if len(saved.Issues) > 0 {
		s.t.Fatalf("flow %q has issues: %+v", name, saved.Issues)
	}
	s.owner.Do("POST", ws+"/automations/"+created.ID+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "finish"}).MustStatus(s.t, 200)
	return uuid.MustParse(created.ID)
}

// newEmail creates an automation email.
func (s *shop) newEmail(name string) string {
	s.t.Helper()
	var e struct{ ID string }
	s.owner.Do("POST", ws+"/automation-emails", map[string]any{"name": name, "subject": name}).MustStatus(s.t, 201).JSON(s.t, &e)
	return e.ID
}

func flowNode(id, typ string, d automations.AutomationNodeData) automations.AutomationNode {
	return automations.AutomationNode{ID: id, Type: typ, Data: d}
}

func flowEdge(src, dst string) automations.AutomationEdge {
	return automations.AutomationEdge{ID: src + "-" + dst, Source: src, Target: dst}
}

// flowRun is a contact's run of a flow.
type flowRun struct {
	Node, Status, Reason string
	Finished             *time.Time
	State                struct {
		Messages map[string]string `json:"messages"`
		Vars     map[string]any    `json:"vars"`
	}
}

func (s *shop) run(flow, contact uuid.UUID) flowRun {
	s.t.Helper()
	var r flowRun
	var state []byte
	s.scan(`SELECT node_id, status, coalesce(exit_reason, ''), finished_at, state FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`,
		[]any{flow, contact}, &r.Node, &r.Status, &r.Reason, &r.Finished, &state)
	_ = json.Unmarshal(state, &r.State)
	return r
}

// "Bought a product" enrolls the buyer, a contact, in the flows of that
// product (or of any product) with the order in Liquid: {{ order.product_name
// }}, {{ order.amount }}, {{ order.number }}, {{ order.currency }}, {{
// order.paid_on }} render in the flow's emails. A buyer who isn't a contact
// enters nothing. A partial refund leaves the contact in the flow; a refund
// in full makes them leave it.
func TestBoughtProductTrigger(t *testing.T) {
	s := newShop(t)
	flows := s.flows()
	email := s.newEmail("Field Guide · thanks")
	trigger := func(product string) automations.AutomationGraph {
		return automations.AutomationGraph{
			Nodes: []automations.AutomationNode{
				flowNode("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerBoughtProduct, ProductID: product}}),
				flowNode("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: email, EmailVersion: 1, Subject: "Your {{ order.product_name }}"}}),
				flowNode("d", automations.NodeDelay, automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: 3, Unit: "days"}}),
				flowNode("x", automations.NodeExit, automations.AutomationNodeData{}),
			},
			Edges: []automations.AutomationEdge{flowEdge("t", "m"), flowEdge("m", "d"), flowEdge("d", "x")},
		}
	}
	guide := s.publishFlow("Field Guide buyers", trigger(s.fieldGuide.String()))
	anyProduct := s.publishFlow("Any product's buyers", trigger(""))
	kit := s.publishFlow("Launch Kit buyers", trigger(s.launchKit.String()))

	const addr = "taylor.reed@example.com"
	taylor := s.addContact(addr, "Taylor")
	o, paid := s.buy(s.fieldGuide, buyer{Email: addr, Name: "Taylor Reed"})
	var payload []byte
	s.scan(`SELECT payload FROM automation_events WHERE workspace_id = $1 AND kind = 'bought_product' AND contact_id = $2 AND ref = $3`,
		[]any{s.ws.ID, taylor, "product:" + s.fieldGuide.String()}, &payload)
	if !strings.Contains(string(payload), o.ID.String()) {
		t.Fatalf("event payload: %s", payload)
	}
	if _, err := flows.Dispatch(s.ctx); err != nil {
		t.Fatal(err)
	}
	if n := s.count(`SELECT count(*) FROM automation_runs WHERE automation_id = $1`, kit); n != 0 {
		t.Errorf("the Launch Kit's flow enrolled %d Field Guide buyers", n)
	}
	want := map[string]string{"product_name": "The Deliverability Field Guide", "amount": "$29.00", "number": fmt.Sprint(o.Number), "currency": "USD",
		"paid_on": o.PaidAt.In(time.UTC).Format("2 Jan 2006")}
	if loc, err := time.LoadLocation(s.ws.Timezone); err == nil && s.ws.Timezone != "" {
		want["paid_on"] = o.PaidAt.In(loc).Format("2 Jan 2006")
	}
	for _, flow := range []uuid.UUID{guide, anyProduct} {
		r := s.run(flow, taylor)
		order, _ := r.State.Vars["order"].(map[string]any)
		for k, v := range want {
			if fmt.Sprint(order[k]) != v {
				t.Errorf("flow %s: {{ order.%s }} = %v, want %s", flow, k, order[k], v)
			}
		}
	}

	// The email step keeps the order with the message, and Liquid renders it
	// as the sending engine does (the contact's variables, then the order).
	if _, err := flows.Tick(s.ctx); err != nil {
		t.Fatal(err)
	}
	r := s.run(guide, taylor)
	msg := r.State.Messages["m"]
	if r.Node != "d" || r.Status != "waiting" || msg == "" {
		t.Fatalf("run after the email step: %+v", r)
	}
	vars := liquid.Vars(addr, "Taylor", "", nil, map[string]string{})
	for k, v := range automations.MessageVars(s.ctx, s.env.App.Q, uuid.MustParse(msg)) {
		if _, taken := vars[k]; !taken {
			vars[k] = v
		}
	}
	tpl, err := engine.Prepare(engine.Content{Subject: "Your {{ order.product_name }}",
		HTML: "<p>Hi {{ first_name }}: {{ order.amount }} · order {{ order.number }} · {{ order.currency }} · paid {{ order.paid_on }}</p>"}, engine.Tracking{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := tpl.Render(vars, false)
	if err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf("Hi Taylor: $29.00 · order %d · USD · paid %s", o.Number, want["paid_on"])
	if out.Subject != "Your The Deliverability Field Guide" || !strings.Contains(out.HTML, line) {
		t.Errorf("rendered: %q\n%s\nwant %q", out.Subject, out.HTML, line)
	}

	// A buyer who isn't a contact enters nothing.
	o2, _ := s.buy(s.fieldGuide, buyer{Email: "not.a.contact@example.com"})
	if n := s.count(`SELECT count(*) FROM automation_events WHERE payload->>'order_id' = $1`, o2.ID.String()); n != 0 {
		t.Errorf("%d automation events for a buyer who isn't a contact", n)
	}

	// Partially refunded: still in the flow. In full: out of both.
	ev, err := s.stripe.Refund(payments.Live, paid.Charge, 900)
	if err != nil {
		t.Fatal(err)
	}
	s.hook(ev)
	if r := s.run(guide, taylor); r.Status != "waiting" {
		t.Errorf("after a partial refund: %+v", r)
	}
	ev, err = s.stripe.Refund(payments.Live, paid.Charge, 0)
	if err != nil {
		t.Fatal(err)
	}
	s.hook(ev)
	for _, flow := range []uuid.UUID{guide, anyProduct} {
		if r := s.run(flow, taylor); r.Status != "exited" || r.Reason != "refunded" || r.Finished == nil {
			t.Errorf("flow %s after the refund: %+v", flow, r)
		}
	}
}

// The goal "Bought a product" moves a contact who buys the product while in
// the flow to the goal; another product, or a purchase refunded in full,
// doesn't.
func TestBoughtProductGoal(t *testing.T) {
	s := newShop(t)
	flows := s.flows()
	const tag = "x19-considering"
	flow := s.publishFlow("Considering the Field Guide", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			flowNode("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerTagAdded, Tag: tag}}),
			flowNode("d", automations.NodeDelay, automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: 5, Unit: "days"}}),
			flowNode("g", automations.NodeGoal, automations.AutomationNodeData{Label: "Bought the Field Guide",
				Condition: &automations.AutomationCondition{Kind: automations.TriggerBoughtProduct, ProductID: s.fieldGuide.String()}}),
		},
		Edges: []automations.AutomationEdge{flowEdge("t", "d"), flowEdge("d", "g")},
	})
	enter := func(email string) uuid.UUID {
		t.Helper()
		id := s.addContact(email, "Reader")
		if err := s.env.App.Q.InsertAutomationEvent(s.ctx, store.InsertAutomationEventParams{WorkspaceID: s.ws.ID, Kind: automations.TriggerTagAdded, ContactID: id,
			Ref: tag, Payload: []byte(`{}`), OccurredAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	buys, other, refunded := enter("goal.buys@example.com"), enter("goal.other@example.com"), enter("goal.refunded@example.com")
	if _, err := flows.Dispatch(s.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.Tick(s.ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range []uuid.UUID{buys, other, refunded} {
		if r := s.run(flow, c); r.Status != "waiting" || r.Node != "d" {
			t.Fatalf("run of %s before buying: %+v", c, r)
		}
	}
	s.buy(s.fieldGuide, buyer{Email: "goal.buys@example.com"})
	s.buy(s.launchKit, buyer{Email: "goal.other@example.com", Amount: 700})
	_, paid := s.buy(s.fieldGuide, buyer{Email: "goal.refunded@example.com"})
	ev, err := s.stripe.Refund(payments.Live, paid.Charge, 0)
	if err != nil {
		t.Fatal(err)
	}
	s.hook(ev)
	if err := flows.Scan(s.ctx); err != nil {
		t.Fatal(err)
	}
	if r := s.run(flow, buys); r.Status != "goal" || r.Node != "g" {
		t.Errorf("the buyer: %+v", r)
	}
	for _, c := range []uuid.UUID{other, refunded} {
		if r := s.run(flow, c); r.Status != "waiting" || r.Node != "d" {
			t.Errorf("run of %s: %+v", c, r)
		}
	}
}
