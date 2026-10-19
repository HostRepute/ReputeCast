package audience

// The publication website in the contacts table (design expansion phase 7):
// the filter "Signed in to the website".

// WebsiteFilter narrows contacts by the website.
type WebsiteFilter struct {
	// SignedIn keeps the members: contacts who signed in to the website.
	SignedIn bool `json:"signedIn,omitempty"`
}

// WebsiteQuery is the same filter as query parameters.
type WebsiteQuery struct {
	SignedIn bool `query:"signedIn" doc:"Only contacts who signed in to the website"`
}

func (q WebsiteQuery) filter() WebsiteFilter { return WebsiteFilter(q) }

// websiteFilter appends the website conditions of f.
func websiteFilter(out *filterSQL, f WebsiteFilter) {
	if f.SignedIn {
		out.add("EXISTS (SELECT 1 FROM website_members wm WHERE wm.contact_id = c.id)")
	}
}
