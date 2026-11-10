package deliverability_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/deliverability"
)

// rfc9990Report is the sample report of RFC 9990 appendix B: the DMARC
// 2.0 namespace, a version, np, testing and discovery_method, no pct, and
// a "pass" disposition.
const rfc9990Report = `<?xml version="1.0" encoding="UTF-8"?>
<feedback xmlns="urn:ietf:params:xml:ns:dmarc-2.0">
  <version>1.0</version>
  <report_metadata>
    <org_name>Sample Reporter</org_name>
    <email>report_sender@example-reporter.com</email>
    <report_id>3v98abbp8ya9n3va8yr8oa3ya</report_id>
    <date_range><begin>302832000</begin><end>302918399</end></date_range>
    <generator>Example DMARC Aggregate Reporter v1.2</generator>
  </report_metadata>
  <policy_published>
    <domain>example.com</domain>
    <p>quarantine</p>
    <sp>none</sp>
    <np>none</np>
    <testing>n</testing>
    <discovery_method>treewalk</discovery_method>
  </policy_published>
  <record>
    <row>
      <source_ip>192.0.2.123</source_ip>
      <count>123</count>
      <policy_evaluated><disposition>pass</disposition><dkim>pass</dkim><spf>fail</spf></policy_evaluated>
    </row>
    <identifiers><envelope_from>example.com</envelope_from><header_from>example.com</header_from></identifiers>
    <auth_results>
      <dkim><domain>example.com</domain><result>pass</result><selector>abc123</selector></dkim>
      <spf><domain>example.com</domain><result>fail</result></spf>
    </auth_results>
  </record>
</feedback>`

func TestParseReportRFC9990(t *testing.T) {
	rep, err := deliverability.ParseReport([]byte(rfc9990Report))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Domain != "example.com" || rep.OrgName != "Sample Reporter" || rep.ReportID != "3v98abbp8ya9n3va8yr8oa3ya" || len(rep.Records) != 1 {
		t.Fatalf("report: %+v", rep)
	}
	want := map[string]string{"domain": "example.com", "p": "quarantine", "sp": "none", "np": "none", "testing": "n", "discovery_method": "treewalk"}
	if !maps.Equal(rep.Policy, want) {
		t.Errorf("policy %v, keys %v", rep.Policy, slices.Sorted(maps.Keys(rep.Policy)))
	}
	r := rep.Records[0]
	// "pass" is RFC 9990's "no action, passed DMARC": delivered, like none.
	if r.Disposition != "none" || r.Count != 123 || r.SourceIP != "192.0.2.123" || r.DKIM != "pass" || r.SPF != "fail" || r.DKIMDomain != "example.com" {
		t.Errorf("record: %+v", r)
	}
}
