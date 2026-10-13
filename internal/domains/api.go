package domains

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type wsInput struct {
	api.WorkspaceInput
}

type senderBody struct {
	api.WorkspaceInput
	Body SenderInput
}

type senderIDInput struct {
	api.WorkspaceInput
	SenderID string `path:"senderId" format:"uuid"`
}

// Huma only reads parameters from exported embedded structs, so every input
// spells out its path parameters.
type senderUpdateInput struct {
	api.WorkspaceInput
	SenderID string `path:"senderId" format:"uuid"`
	Body     SenderInput
}

type defaultsInput struct {
	api.WorkspaceInput
	Body SenderDefaultsInput
}

type domainBody struct {
	api.WorkspaceInput
	Body DomainInput
}

type domainIDInput struct {
	api.WorkspaceInput
	DomainID string `path:"domainId" format:"uuid"`
}

type domainGetInput struct {
	api.WorkspaceInput
	DomainID string `path:"domainId" format:"uuid"`
	Provider string `query:"provider" doc:"Only the records this provider needs (and the domain-wide ones)"`
	Refresh  bool   `query:"refresh" doc:"Check DNS again first when the last check is older than 20 seconds"`
}

type domainUpdateInput struct {
	api.WorkspaceInput
	DomainID string `path:"domainId" format:"uuid"`
	Body     UpdateDomainInput
}

type domainAttachInput struct {
	api.WorkspaceInput
	DomainID   string `path:"domainId" format:"uuid"`
	ProviderID string `path:"providerId" format:"uuid"`
}

// CloudflareInput adds records through the Cloudflare API.
type CloudflareInput struct {
	Token string `json:"token" minLength:"20" maxLength:"200" doc:"A Cloudflare API token with Zone › DNS › Edit on the domain's zone. Used once, never stored."`
}

type cloudflareBody struct {
	api.WorkspaceInput
	DomainID string `path:"domainId" format:"uuid"`
	Body     CloudflareInput
}

// SendingDomainCreated is a new domain with what to do in providers'
// dashboards that ReputeCast couldn't set up.
type SendingDomainCreated struct {
	Domain SendingDomainDetail `json:"domain"`
	Notes  []ProviderSetupNote `json:"notes"`
}

// SendingDomainCloudflareResult is what Cloudflare changed and the fresh check.
type SendingDomainCloudflareResult struct {
	Result CloudflareResult    `json:"result"`
	Domain SendingDomainDetail `json:"domain"`
}

// Register adds the senders and domains operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Senders & domains"}
	view, manage := api.Workspace(authz.ProvidersView), api.Workspace(authz.DomainsManage)

	api.Get(h, api.Op{ID: "list-senders", Path: "/workspaces/{workspace}/senders", Summary: "Sender identities, sending domains and sending defaults",
		Description: "Verification is per sending provider: each provider signs with its own DKIM key.", Tags: tags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[SendersOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-sender", Path: "/workspaces/{workspace}/senders", Summary: "Add a sender identity on a sending domain", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *senderBody) (*api.Body[SenderIdentity], error) {
			out, err := s.CreateSender(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-sender", Path: "/workspaces/{workspace}/senders/{senderId}", Summary: "Change a sender's name, reply-to, default streams or providers", Tags: tags, Access: manage},
		func(ctx context.Context, in *senderUpdateInput) (*api.Body[SenderIdentity], error) {
			out, err := s.UpdateSender(ctx, api.WorkspaceFrom(ctx), in.SenderID, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-sender", Path: "/workspaces/{workspace}/senders/{senderId}", Summary: "Delete a sender identity",
		Description: "Unsent campaigns that used it must pick another From before they send.", Tags: tags, Access: manage},
		func(ctx context.Context, in *senderIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteSender(ctx, api.WorkspaceFrom(ctx), in.SenderID)
		})
	api.Put(h, api.Op{ID: "save-sender-defaults", Path: "/workspaces/{workspace}/senders/defaults", Summary: "Save the default From, reply handling, footer address and link tracking domain",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *defaultsInput) (*api.Body[SenderDefaults], error) {
			out, err := s.SaveDefaults(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-sending-domain", Path: "/workspaces/{workspace}/domains", Summary: "Add a sending domain and fetch the records every provider needs",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *domainBody) (*api.Body[SendingDomainCreated], error) {
			ws := api.WorkspaceFrom(ctx)
			d, notes, err := s.CreateDomain(ctx, ws, in.Body)
			if err != nil {
				return nil, err
			}
			det, err := s.detailOf(ctx, ws, d, "")
			return api.OK(SendingDomainCreated{Domain: det, Notes: notes}), err
		})
	api.Get(h, api.Op{ID: "get-sending-domain", Path: "/workspaces/{workspace}/domains/{domainId}", Summary: "A sending domain's DNS records with their live status",
		Tags: tags, Access: view},
		func(ctx context.Context, in *domainGetInput) (*api.Body[SendingDomainDetail], error) {
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), in.DomainID, in.Provider, in.Refresh)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-sending-domain", Path: "/workspaces/{workspace}/domains/{domainId}", Summary: "Rename a sending domain's note or set its tracking domain",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *domainUpdateInput) (*api.Body[SendingDomainDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			d, err := s.UpdateDomain(ctx, ws, in.DomainID, in.Body)
			if err != nil {
				return nil, err
			}
			out, err := s.detailOf(ctx, ws, d, "")
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-sending-domain", Path: "/workspaces/{workspace}/domains/{domainId}", Summary: "Delete a sending domain no sender uses", Tags: tags, Access: manage},
		func(ctx context.Context, in *domainIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteDomain(ctx, api.WorkspaceFrom(ctx), in.DomainID)
		})
	api.Post(h, api.Op{ID: "check-sending-domain", Path: "/workspaces/{workspace}/domains/{domainId}/check", Summary: "Check a sending domain's DNS now", Tags: tags, Access: manage},
		func(ctx context.Context, in *domainIDInput) (*api.Body[SendingDomainDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			id, err := parseID(in.DomainID, "Sending domain")
			if err != nil {
				return nil, err
			}
			d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return nil, api.MapError(err, "Sending domain")
			}
			if d, err = s.Check(ctx, d); err != nil {
				return nil, err
			}
			if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "domain.check", TargetType: "sending_domain", TargetID: d.ID.String(), TargetName: d.Domain,
				Metadata: map[string]any{"status": d.Status}}); err != nil {
				return nil, err
			}
			out, err := s.detailOf(ctx, ws, d, "")
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "attach-sending-domain-provider", Path: "/workspaces/{workspace}/domains/{domainId}/providers/{providerId}",
		Summary: "Fetch the records a provider needs for the domain", Tags: tags, Access: manage},
		func(ctx context.Context, in *domainAttachInput) (*api.Body[SendingDomainDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			did, err := parseID(in.DomainID, "Sending domain")
			if err != nil {
				return nil, err
			}
			pid, err := parseID(in.ProviderID, "Provider")
			if err != nil {
				return nil, err
			}
			note, err := s.AttachProvider(ctx, ws, did, pid)
			if err != nil {
				return nil, err
			}
			d, err := s.Q.DomGetAny(ctx, did)
			if err != nil {
				return nil, err
			}
			out, err := s.detailOf(ctx, ws, d, "")
			if note != "" {
				out.Notes = append(out.Notes, ProviderSetupNote{ProviderID: in.ProviderID, Text: note})
			}
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-sending-domain-records-cloudflare", Path: "/workspaces/{workspace}/domains/{domainId}/cloudflare",
		Summary: "Add the missing records through the Cloudflare API", Description: "The API token is used for this request only and never stored.",
		Tags: tags, Access: api.WorkspaceHuman(authz.DomainsManage)},
		func(ctx context.Context, in *cloudflareBody) (*api.Body[SendingDomainCloudflareResult], error) {
			ws := api.WorkspaceFrom(ctx)
			out, err := s.AddViaCloudflare(ctx, ws, in.DomainID, in.Body.Token)
			return api.OK(out), err
		})
}

// AddViaCloudflare creates or updates the domain's unverified records in
// its Cloudflare zone, then checks DNS again.
func (s *Service) AddViaCloudflare(ctx context.Context, ws *store.Workspace, rawID, token string) (SendingDomainCloudflareResult, error) {
	id, err := parseID(rawID, "Sending domain")
	if err != nil {
		return SendingDomainCloudflareResult{}, err
	}
	d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SendingDomainCloudflareResult{}, api.MapError(err, "Sending domain")
	}
	recs := DecodeRecords(d.DnsRecords)
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	res, err := s.Cloudflare.Apply(cctx, strings.TrimSpace(token), d.Domain, recs)
	cancel()
	if errors.Is(err, ErrCloudflareToken) {
		return SendingDomainCloudflareResult{}, api.Invalid("body.token", "Cloudflare rejected this token. Use an API token with Zone › DNS › Edit permission on "+d.Domain+"'s zone.")
	}
	if err != nil {
		return SendingDomainCloudflareResult{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, err.Error()+". Nothing else was changed.")
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "domain.dns_cloudflare", TargetType: "sending_domain", TargetID: d.ID.String(), TargetName: d.Domain,
		After: map[string]any{"zone": res.Zone, "created": res.Created, "updated": res.Updated, "failed": res.Failed}}); err != nil {
		return SendingDomainCloudflareResult{}, err
	}
	// Cloudflare answers within seconds; check right away.
	if checked, err := s.Check(ctx, d); err == nil {
		d = checked
	}
	det, err := s.detailOf(ctx, ws, d, "")
	return SendingDomainCloudflareResult{Result: res, Domain: det}, err
}

// SenderByID is a sender of the workspace (system mail, the connect flow).
func (s *Service) SenderByID(ctx context.Context, wsID, id uuid.UUID) (store.SenderGetRow, error) {
	return s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: wsID, ID: id})
}
