package assistant_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
)

// A thread's model is the one the person picked for the assistant (their
// own key's included), or else the workspace's. Someone else's personal
// model isn't there, and a model whose tool check didn't pass can't hold a
// thread.
func TestAssistantModelsFollowThePerson(t *testing.T) {
	f := newFixture(t)
	f.oc.Do(http.MethodPut, f.base+"/personal-keys", map[string]any{"enabled": true, "capUsd": 10}).MustStatus(t, 200)
	var own ai.AIPersonalSettings
	f.mc.Do(http.MethodPost, f.base+"/personal/accounts", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "own-model",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &own)
	account, model := own.Accounts[0].ID, own.Accounts[0].Models[0].ID
	f.passTools(f.mc, "/personal/accounts/"+account.String()+"/models/"+model.String())
	f.mc.Do(http.MethodPut, f.base+"/personal/features/assistant/model", map[string]any{"modelId": model}).MustStatus(t, 200)

	if th := f.thread(f.mc, map[string]any{}); th.Model == nil || th.Model.ID != model || !th.Model.Personal || !th.Model.Usable {
		t.Fatalf("the member's thread is on %+v, want their own model %s", th.Model, model)
	}
	if th := f.thread(f.oc, map[string]any{}); th.Model == nil || th.Model.ID != f.modelID || th.Model.Personal {
		t.Fatalf("the owner's thread is on %+v, want the workspace's %s", th.Model, f.modelID)
	}
	// Someone else's personal model isn't there.
	f.cc.Do(http.MethodPost, f.base+"/assistant/threads", map[string]any{"modelId": model, "requestKey": "colleague-1"}).MustStatus(t, 404)
	// A model whose tool check didn't pass (or that isn't the workspace's) can't hold a thread.
	var s ai.AISettings
	f.oc.Do(http.MethodPost, f.base+"/accounts/"+f.accountID.String()+"/models", map[string]any{"model": "model-b", "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &s)
	unchecked := s.Accounts[0].Models[1].ID
	if p := f.mc.Do(http.MethodPost, f.base+"/assistant/threads", map[string]any{"modelId": unchecked, "requestKey": "unchecked-1"}).MustStatus(t, 409).Problem(t); p["code"] != "ai_tools_unchecked" {
		t.Fatalf("a thread on an unchecked model: %v", p)
	}
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads", map[string]any{"modelId": uuid.New(), "requestKey": "missing-1"}).MustStatus(t, 404)

	// The picker: the workspace's models and the caller's own, each with why
	// it can't be used.
	var list ai.AIAssistantModelList
	f.mc.Do(http.MethodGet, f.base+"/assistant/models", nil).MustStatus(t, 200).JSON(t, &list)
	byID := map[uuid.UUID]ai.AIAssistantModel{}
	for _, m := range list.Models {
		byID[m.ID] = m
	}
	if list.DefaultID == nil || *list.DefaultID != model || len(list.Models) != 3 || !byID[model].Personal || !byID[f.modelID].Usable ||
		byID[unchecked].Usable || byID[unchecked].Reason != "tools_unchecked" {
		t.Fatalf("the member's picker: %+v", list)
	}
	f.cc.Do(http.MethodGet, f.base+"/assistant/models", nil).MustStatus(t, 200).JSON(t, &list)
	if list.DefaultID == nil || *list.DefaultID != f.modelID || len(list.Models) != 2 {
		t.Fatalf("a colleague's picker: %+v", list)
	}
	// With personal keys off, the member's own pick can't hold a thread.
	f.oc.Do(http.MethodPut, f.base+"/personal-keys", map[string]any{"enabled": false, "capUsd": 10}).MustStatus(t, 200)
	if p := f.mc.Do(http.MethodPost, f.base+"/assistant/threads", map[string]any{"requestKey": "off-1"}).MustStatus(t, 409).Problem(t); p["code"] != "ai_personal_keys_off" {
		t.Fatalf("a thread on a personal key while they are off: %v", p)
	}
}
