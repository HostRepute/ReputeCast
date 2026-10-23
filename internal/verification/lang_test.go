package verification_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/verification"
)

// TestVerificationTextInVietnamese checks that a member reading Vietnamese
// gets verification's server text in Vietnamese: a problem made for the
// request, and a job started in an English request, named by its tier, with
// the service's last use and the jobs tray.
func TestVerificationTextInVietnamese(t *testing.T) {
	f := setup(t, map[string]string{"zerobounce": httptest.NewServer(fakeZeroBounce()).URL})
	f.contacts(t, "sunset", "valid1@x.test", "invalid1@x.test")
	f.connect(t, "zerobounce", 0.008, true, false)
	vi := i18n.WithLang(f.ctx, i18n.For(i18n.Vietnamese))

	// A problem made for the request.
	_, err := f.svc.Estimate(vi, &f.ws, verification.VerificationEstimateInput{Target: verification.VerificationTarget{Kind: "tier"}})
	if b, _ := json.Marshal(err); err == nil || !strings.Contains(string(b), "Hãy chọn một nhóm.") {
		t.Fatalf("Vietnamese problem: %s", b)
	}

	// A job started in English reads in Vietnamese, and still in English.
	j, err := f.svc.Start(f.ctx, &f.ws, verification.VerificationStartInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"sunset"}}, Origin: "list_health"})
	if err != nil {
		t.Fatal(err)
	}
	en, _ := f.svc.Job(f.ctx, &f.ws, uuid.MustParse(j.ID))
	vn, _ := f.svc.Job(vi, &f.ws, uuid.MustParse(j.ID))
	if en.Label != "Sunset" || vn.Label != "Ngừng gửi dần" || vn.RegionLabel != "Hoa Kỳ" {
		t.Fatalf("job label: %q, %q, region %q", en.Label, vn.Label, vn.RegionLabel)
	}
	o, err := f.svc.Overview(vi, &f.ws)
	if err != nil {
		t.Fatal(err)
	}
	var zb *verification.VerificationServiceView
	for i := range o.Services {
		if o.Services[i].Key == "zerobounce" {
			zb = &o.Services[i]
		}
	}
	if zb == nil || zb.LastUsedWhat != "nhóm Ngừng gửi dần" || zb.LastSuccessWhat != "tác vụ nhóm Ngừng gửi dần" {
		t.Fatalf("last use: %+v", zb)
	}
	if len(o.Uses) == 0 || o.Uses[0].Label != "Ngừng gửi dần" {
		t.Fatalf("uses: %+v", o.Uses)
	}
	tray, err := f.svc.TrayJobs(vi, &f.ws)
	if err != nil || len(tray) != 1 || tray[0].Title != "Đã xác minh nhóm Ngừng gửi dần bằng ZeroBounce" {
		t.Fatalf("tray: %+v, %v", tray, err)
	}
}
