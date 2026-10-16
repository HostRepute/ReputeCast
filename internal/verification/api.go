package verification

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
)

type serviceInput struct {
	api.WorkspaceInput
	Service string `path:"service" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable"`
}

type connectInput struct {
	api.WorkspaceInput
	Body VerificationConnectInput
}

type testKeyInput struct {
	api.WorkspaceInput
	Body VerificationKeyInput
}

type settingsInput struct {
	serviceInput
	Body VerificationSettingsInput
}

type replaceKeyInput struct {
	serviceInput
	Body VerificationReplaceKeyInput
}

type disconnectInput struct {
	serviceInput
	Forms []string `query:"forms" maxItems:"50" doc:"The forms that stop checking, as the dialog listed them"`
}

type estimateInput struct {
	api.WorkspaceInput
	Body VerificationEstimateInput
}

type startInput struct {
	api.WorkspaceInput
	Body VerificationStartInput
}

type jobInput struct {
	api.WorkspaceInput
	Job string `path:"job" format:"uuid"`
}

type contactInput struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid"`
}

type verifyContactInput struct {
	contactInput
	Body VerifyContactInput
}

type formInput struct {
	api.WorkspaceInput
	Form string `path:"form" pattern:"^[a-z0-9_-]{1,64}$" doc:"A form's id, or subscribe_page"`
}

type setFormInput struct {
	formInput
	Name string `query:"name" maxLength:"200" doc:"The form's name, for the audit log"`
	Body VerificationFormInput
}

type csvOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}

func id(raw, what string) (uuid.UUID, error) {
	u, err := uuid.Parse(raw)
	if err != nil {
		return u, api.NotFound(what)
	}
	return u, nil
}

// Register adds the verification operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Verification"}
	base := "/workspaces/{workspace}/verification"

	api.Get(h, api.Op{ID: "list-verification-services", Path: base + "/services", Summary: "Verification services, their connections and uses",
		Tags: tags, Access: api.Workspace(authz.IntegrationsView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[VerificationOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-verification-key", Path: base + "/services/test", Summary: "Test a service's API key by reading its balance (no address is sent)",
		Tags: tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, in *testKeyInput) (*api.Body[VerificationKeyTest], error) {
			out, err := s.TestKey(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-verification-service", Path: base + "/services", Summary: "Connect a verification service with the workspace's own API key",
		Tags: tags, Access: api.Workspace(authz.IntegrationsManage), Status: http.StatusCreated},
		func(ctx context.Context, in *connectInput) (*api.Body[VerificationServiceView], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-verification-service", Path: base + "/services/{service}", Summary: "One verification service",
		Tags: tags, Access: api.Workspace(authz.IntegrationsView)},
		func(ctx context.Context, in *serviceInput) (*api.Body[VerificationServiceView], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), in.Service)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-verification-service", Path: base + "/services/{service}", Summary: "Change a service's price per check, region and default uses",
		Tags: tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, in *settingsInput) (*api.Body[VerificationServiceView], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), in.Service, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-verification-key", Path: base + "/services/{service}/key", Summary: "Replace a service's API key (write-only)",
		Tags: tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, in *replaceKeyInput) (*api.Body[VerificationServiceView], error) {
			out, err := s.ReplaceKey(ctx, api.WorkspaceFrom(ctx), in.Service, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-verification-service", Path: base + "/services/{service}/test", Summary: "Read a connected service's balance again (no address is sent)",
		Tags: tags, Access: api.Workspace(authz.IntegrationsView)},
		func(ctx context.Context, in *serviceInput) (*api.Body[VerificationKeyTest], error) {
			out, err := s.TestConnection(ctx, api.WorkspaceFrom(ctx), in.Service)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-verification-service", Path: base + "/services/{service}", Summary: "Disconnect a service; results already on contacts stay",
		Tags: tags, Access: api.Workspace(authz.IntegrationsManage), Status: http.StatusOK},
		func(ctx context.Context, in *disconnectInput) (*api.Body[VerificationDisconnectResult], error) {
			out, err := s.Disconnect(ctx, api.WorkspaceFrom(ctx), in.Service, in.Forms)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-verification-realtime", Path: base + "/services/{service}/realtime", Summary: "A service's real-time checks by form and by day",
		Tags: tags, Access: api.Workspace(authz.IntegrationsView)},
		func(ctx context.Context, in *serviceInput) (*api.Body[VerificationRealtime], error) {
			out, err := s.Realtime(ctx, api.WorkspaceFrom(ctx), in.Service)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "estimate-verification", Path: base + "/estimate", Summary: "What a use would check and cost, for its dialog and notice",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *estimateInput) (*api.Body[VerificationEstimate], error) {
			out, err := s.Estimate(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-verification", Path: base + "/jobs", Summary: "Send addresses to the verification service and start a job (audited)",
		Tags: tags, Access: api.Workspace(authz.SubscribersVerify), Status: http.StatusAccepted},
		func(ctx context.Context, in *startInput) (*api.Body[VerificationJobView], error) {
			out, err := s.Start(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-verification-job", Path: base + "/jobs/{job}", Summary: "A verification job's progress and results",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *jobInput) (*api.Body[VerificationJobView], error) {
			jid, err := id(in.Job, "Verification job")
			if err != nil {
				return nil, err
			}
			out, err := s.Job(ctx, api.WorkspaceFrom(ctx), jid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "cancel-verification-job", Path: base + "/jobs/{job}/cancel", Summary: "Stop a job; the results so far are kept",
		Tags: tags, Access: api.Workspace(authz.SubscribersVerify)},
		func(ctx context.Context, in *jobInput) (*api.Body[VerificationJobView], error) {
			jid, err := id(in.Job, "Verification job")
			if err != nil {
				return nil, err
			}
			out, err := s.Cancel(ctx, api.WorkspaceFrom(ctx), jid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-verification-job", Path: base + "/jobs/{job}/resume", Summary: "Continue a job that stopped for credits or an outage",
		Tags: tags, Access: api.Workspace(authz.SubscribersVerify)},
		func(ctx context.Context, in *jobInput) (*api.Body[VerificationJobView], error) {
			jid, err := id(in.Job, "Verification job")
			if err != nil {
				return nil, err
			}
			out, err := s.Resume(ctx, api.WorkspaceFrom(ctx), jid)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "download-verification-results", Path: base + "/jobs/{job}/results.csv", Summary: "A job's results as CSV: addresses, so it's an export (audited)",
		Tags: tags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *jobInput) (*csvOutput, error) {
			jid, err := id(in.Job, "Verification job")
			if err != nil {
				return nil, err
			}
			b, name, err := s.ResultsCSV(ctx, api.WorkspaceFrom(ctx), jid)
			if err != nil {
				return nil, err
			}
			return &csvOutput{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, Body: b}, nil
		})

	api.Get(h, api.Op{ID: "get-contact-address-check", Path: base + "/contacts/{contact}", Summary: "A contact's latest address check and what Verify now would use",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *contactInput) (*api.Body[ContactAddressCheck], error) {
			cid, err := id(in.Contact, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactCheck(ctx, api.WorkspaceFrom(ctx), cid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "verify-contact", Path: base + "/contacts/{contact}/verify", Summary: "Check one contact's address now (audited)",
		Tags: tags, Access: api.Workspace(authz.SubscribersVerify)},
		func(ctx context.Context, in *verifyContactInput) (*api.Body[ContactAddressCheck], error) {
			cid, err := id(in.Contact, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.VerifyContact(ctx, api.WorkspaceFrom(ctx), cid, in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-form-address-check", Path: base + "/forms/{form}", Summary: "A form's real-time address check and its 30-day numbers",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *formInput) (*api.Body[VerificationFormSetting], error) {
			out, err := s.FormSetting(ctx, api.WorkspaceFrom(ctx), in.Form)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-form-address-check", Path: base + "/forms/{form}", Summary: "Turn a form's real-time address check on or off (also needs forms:edit; audited)",
		Tags: tags, Access: api.Workspace(authz.SubscribersVerify)},
		func(ctx context.Context, in *setFormInput) (*api.Body[VerificationFormSetting], error) {
			out, err := s.SetFormCheck(ctx, api.WorkspaceFrom(ctx), in.Form, in.Name, in.Body)
			return api.OK(out), err
		})
}

// RegisterJobs adds the workers and the periodic jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &JobWorker{Svc: s})
	river.AddWorker(reg.Workers, &SweepWorker{Svc: s})
	river.AddWorker(reg.Workers, &BalanceWorker{Svc: s})
	river.AddWorker(reg.Workers, &PendingWorker{Svc: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return BalanceArgs{}, nil }, false)
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return PendingArgs{}, nil }, false)
}
