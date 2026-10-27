package ses_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/ses"
)

// PR-11: each workspace sends as its own SES tenant, created with its own
// suppression list (scope TENANT, bounces and complaints); the reputation
// policy SES applies to the tenant is read back for the setup's note.
func TestTenantProvisioning(t *testing.T) {
	var mu sync.Mutex
	calls := []string{}
	var created struct {
		TenantName            string
		SuppressionAttributes struct {
			SuppressionScope  string
			SuppressedReasons []string
		}
	}
	exists := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/email/tenants":
			if exists {
				w.Header().Set("X-Amzn-ErrorType", "AlreadyExistsException")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"__type":"AlreadyExistsException","message":"Tenant already exists"}`)
				return
			}
			_ = json.Unmarshal(body, &created)
			_, _ = io.WriteString(w, `{"TenantName":"`+created.TenantName+`","TenantArn":"arn:aws:ses:us-east-1:123456789012:tenant/`+created.TenantName+`/t-1"}`)
		case r.URL.Path == "/v2/email/tenant/suppression":
			_, _ = io.WriteString(w, `{}`)
		case r.URL.Path == "/v2/email/tenants/get":
			_, _ = io.WriteString(w, `{"Tenant":{"TenantName":"acme-newsletter","TenantArn":"arn:aws:ses:us-east-1:123456789012:tenant/acme-newsletter/t-1"}}`)
		case strings.HasPrefix(r.URL.Path, "/v2/email/reputation/entities/"):
			_, _ = io.WriteString(w, `{"ReputationEntity":{"ReputationManagementPolicy":"arn:aws:ses:us-east-1:aws:reputation-policy/standard"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"__type":"NotFoundException","message":"unexpected `+r.URL.Path+`"}`)
		}
	}))
	defer srv.Close()
	settings, _ := json.Marshal(ses.Settings{Region: "us-east-1", Access: "keys", Endpoint: srv.URL})
	p, err := ses.New(providers.Config{Settings: settings, HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Credentials: json.RawMessage(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`)})
	if err != nil {
		t.Fatal(err)
	}
	pv := p.(providers.Provisioner)
	in := providers.ProvisionInput{Tenant: "Acme Newsletter!"}
	steps := pv.ProvisionSteps(context.Background(), in)
	if steps[0].Key != "tenant" {
		t.Fatalf("the tenant comes first: %+v", steps)
	}
	res, err := pv.Provision(context.Background(), "tenant", in)
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantName != "Acme-Newsletter" || created.SuppressionAttributes.SuppressionScope != "TENANT" ||
		strings.Join(created.SuppressionAttributes.SuppressedReasons, ",") != "BOUNCE,COMPLAINT" {
		t.Fatalf("created %+v", created)
	}
	if res.Settings["tenant"] != "Acme-Newsletter" || res.Settings["tenant_reputation_policy"] != "standard" || !strings.Contains(res.Note, "Standard") {
		t.Fatalf("result %+v", res)
	}

	// A tenant from an earlier setup keeps (gets) its own list too.
	exists = true
	if _, err := pv.Provision(context.Background(), "tenant", in); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !contains(calls, "POST /v2/email/tenant/suppression") {
		t.Fatalf("calls %v", calls)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
