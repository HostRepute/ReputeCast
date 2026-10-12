package accounts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// SetupState drives /setup: the first sign-in of the install Owner
// (A5 §5.5), or the steps anyone else must finish before using the app
// (replace a temporary password, add a required second factor).
type SetupState struct {
	// Mode: instance (first Owner), account (personal steps only) or done.
	Mode    string      `json:"mode" enum:"instance,account,done"`
	Email   string      `json:"email"`
	Name    string      `json:"name"`
	Steps   []SetupStep `json:"steps"`
	Current string      `json:"current" doc:"Key of the first unfinished step"`
	// Next is where to go when Mode is done.
	Next          string             `json:"next,omitempty"`
	Password      SetupPassword      `json:"password"`
	SecondFactor  SetupSecondFactor  `json:"secondFactor"`
	RecoveryCodes SetupRecoveryCodes `json:"recoveryCodes"`
	Instance      SetupInstance      `json:"instance"`
	Workspace     *CreatedWorkspace  `json:"workspace,omitempty"`
}

// SetupStep is one step of the stepper.
type SetupStep struct {
	Key   string `json:"key" enum:"password,second_factor,recovery_codes,instance,workspace,provider"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

// SetupPassword is the temporary-password step.
type SetupPassword struct {
	MustChange    bool `json:"mustChange"`
	MinLength     int  `json:"minLength"`
	BreachedCheck bool `json:"breachedCheck"`
}

// SetupSecondFactor is the passkey or authenticator step.
type SetupSecondFactor struct {
	TOTP     bool     `json:"totp"`
	Passkeys []string `json:"passkeys" doc:"Names of the passkeys added"`
	Required bool     `json:"required"`
	Reason   string   `json:"reason,omitempty"`
	// GraceUntil is set while a required second factor can still wait.
	GraceUntil *time.Time `json:"graceUntil,omitempty"`
}

// SetupRecoveryCodes is the recovery codes step.
type SetupRecoveryCodes struct {
	Remaining int        `json:"remaining"`
	SavedAt   *time.Time `json:"savedAt,omitempty"`
}

// SetupInstance is the instance basics step.
type SetupInstance struct {
	Name        string `json:"name"`
	PublicURL   string `json:"publicUrl"`
	Timezone    string `json:"timezone"`
	UpdateCheck bool   `json:"updateCheck"`
	Confirmed   bool   `json:"confirmed"`
	Version     string `json:"version"`
}

// Setup returns the setup steps for the signed-in person.
func (s *Service) Setup(ctx context.Context) (SetupState, error) {
	cu, err := currentUser(ctx)
	if err != nil {
		return SetupState{}, err
	}
	u := *cu
	st, err := s.policy(ctx)
	if err != nil {
		return SetupState{}, err
	}
	keys, err := s.Q.ListPasskeys(ctx, u.ID)
	if err != nil {
		return SetupState{}, err
	}
	codes, err := s.Q.CountUnusedRecoveryCodes(ctx, u.ID)
	if err != nil {
		return SetupState{}, err
	}
	out := SetupState{
		Email: u.Email, Name: u.Name,
		Password:      SetupPassword{MustChange: u.MustChangePassword, MinLength: int(st.PasswordMinLength), BreachedCheck: st.BreachedCheck},
		SecondFactor:  SetupSecondFactor{TOTP: u.TotpEnabledAt != nil, Passkeys: []string{}},
		RecoveryCodes: SetupRecoveryCodes{Remaining: int(codes), SavedAt: u.RecoveryCodesSavedAt},
		Instance: SetupInstance{Name: st.Name, PublicURL: st.PublicUrl, Timezone: st.DefaultTimezone, UpdateCheck: st.UpdateCheck,
			Confirmed: st.BasicsConfirmedAt != nil, Version: version.Get().Version},
		Steps: []SetupStep{},
	}
	if out.Instance.PublicURL == "" {
		out.Instance.PublicURL = s.publicURL(ctx)
	}
	for _, k := range keys {
		out.SecondFactor.Passkeys = append(out.SecondFactor.Passkeys, k.Name)
	}
	hasSecond := out.SecondFactor.TOTP || len(keys) > 0
	codesSaved := codes > 0 && u.RecoveryCodesSavedAt != nil
	if wss, err := s.Q.ListUserWorkspaces(ctx, u.ID); err == nil && len(wss) > 0 {
		out.Workspace = &CreatedWorkspace{ID: wss[0].ID.String(), Slug: wss[0].Slug, Name: wss[0].Name}
	}

	if u.IsInstanceAdmin && st.SetupCompletedAt == nil {
		out.Mode = "instance"
		out.SecondFactor.Required, out.SecondFactor.Reason = true, "Two-factor is required for the first Owner; it can't be skipped."
		out.Steps = []SetupStep{
			{Key: "password", Label: "Replace the temporary password", Done: !u.MustChangePassword},
			{Key: "second_factor", Label: "Add a passkey or authenticator", Done: hasSecond},
			{Key: "recovery_codes", Label: "Save your recovery codes", Done: hasSecond && codesSaved},
			{Key: "instance", Label: "Instance basics", Done: st.BasicsConfirmedAt != nil},
			{Key: "workspace", Label: "Create the first workspace", Done: out.Workspace != nil},
			{Key: "provider", Label: "Connect a sending provider"},
		}
	} else {
		required, reason := s.twoFactorRequired(ctx, u, st)
		needsSecond := required && !hasSecond && !(u.TwofaGraceUntil != nil && u.TwofaGraceUntil.After(s.now()))
		if sess := api.SessionFrom(ctx); sess != nil && sess.MfaVerified && !hasSecond && u.PasswordHash == nil {
			needsSecond = false // SSO sign-ins where the identity provider asserted MFA
		}
		out.SecondFactor.Required, out.SecondFactor.Reason = required, reason
		if required && !hasSecond {
			out.SecondFactor.GraceUntil = u.TwofaGraceUntil
		}
		if u.MustChangePassword {
			out.Steps = append(out.Steps, SetupStep{Key: "password", Label: "Replace the temporary password"})
		}
		if needsSecond {
			out.Steps = append(out.Steps,
				SetupStep{Key: "second_factor", Label: "Add a passkey or authenticator"},
				SetupStep{Key: "recovery_codes", Label: "Save your recovery codes"})
		}
		out.Mode = "account"
		if len(out.Steps) == 0 {
			out.Mode = "done"
			out.Next = s.nextPath(ctx, u)
		}
	}
	for _, st := range out.Steps {
		if !st.Done {
			out.Current = st.Key
			break
		}
	}
	return out, nil
}

// MarkRecoveryCodesSaved records "I have saved these codes".
func (s *Service) MarkRecoveryCodesSaved(ctx context.Context) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	n, err := s.Q.CountUnusedRecoveryCodes(ctx, u.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return api.Conflict("There are no recovery codes to save. Generate them first.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetRecoveryCodesSaved(ctx, u.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.recovery_codes_saved", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Metadata: map[string]any{"codes": n}})
	})
}
