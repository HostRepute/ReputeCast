package website

import (
	"context"
	"errors"
	"html"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The account page's Comments section (M61, PublicAccount): the name a
// member's comments show and "Email me when someone replies to my
// comments". It shows while the site takes comments, or once the member
// has a display name. The form posts to account/comments (internal/
// publicpages, SaveCommentSettings).

func init() { RegisterAccountSection(commentsAccountSection) }

func commentsAccountSection(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, l i18n.Lang) (AccountSection, bool, error) {
	q := AccountQueries(ctx)
	if q == nil {
		return AccountSection{}, false, nil
	}
	set, err := q.WebsiteSettingsGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountSection{}, false, nil
	}
	if err != nil {
		return AccountSection{}, false, err
	}
	who, err := q.WebsiteCommenter(ctx, store.WebsiteCommenterParams{WorkspaceID: ws.ID, ContactID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountSection{}, false, nil
	}
	if err != nil {
		return AccountSection{}, false, err
	}
	if (set.Comments == CommentsOff || set.Comments == "") && strings.TrimSpace(who.DisplayName) == "" {
		return AccountSection{}, false, nil
	}
	w := WordsIn(l)
	t := func(key string) string { return html.EscapeString(w.Text(key, nil)) }
	name := strings.TrimSpace(who.DisplayName)
	if name == "" {
		name = CommenterName("", who.FirstName, who.LastName)
	}
	checked := ""
	if who.CommentEmails {
		checked = " checked"
	}
	var b strings.Builder
	b.WriteString(`<h2 id="acm-h">` + t("comments.account.title") + `</h2>`)
	b.WriteString(`<form method="post" action="account/comments" class="st-stack">`)
	b.WriteString(`<div class="st-stack st-acm"><label class="st-lbl">` + t("comments.nameLabel") + `<input class="st-in" type="text" name="display_name" value="` +
		html.EscapeString(name) + `" maxlength="60" autocomplete="nickname" aria-describedby="acm-name-h"></label>`)
	b.WriteString(`<span class="st-k" id="acm-name-h">` + t("comments.account.nameHint") + `</span></div>`)
	b.WriteString(`<div class="st-line"><div class="tx"><b id="acm-emails">` + t("comments.account.emails") + `</b><span class="st-k">` + t("comments.account.emailsHint") +
		`</span></div><input type="checkbox" class="st-sw" role="switch" name="emails" value="on"` + checked + ` aria-labelledby="acm-emails"></div>`)
	b.WriteString(`<div class="st-row2"><button class="st-btn sm" type="submit">` + t("comments.account.save") + `</button></div></form>`)
	return AccountSection{ID: "comments", Title: w.Text("comments.account.title", nil), HTML: b.String()}, true, nil
}
