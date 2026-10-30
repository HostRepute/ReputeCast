package readiness

// retryAudience is the Audience quality line of a retry (review AN-7):
// whom it sends the issue to again, and what to make sure of first.
func (b *builder) retryAudience() {
	a := b.f.Audience
	r := a.Retry
	if r == nil || a.Empty {
		return
	}
	args := A{"issue": r.Issue, "count": a.Final, "blocked": r.Blocked}
	c := check{Key: "audience.retry", Group: GroupAudience, Status: Pass, Label: tx("label.retry", nil),
		Title: tx("retry.title", args), Short: tx("retry.short", nil), Detail: tx("retry.detail", args),
		Fixes: []fix{link(tx("fix.openDeliverability", nil), b.href("/deliverability"))}}
	b.add(c)
}
