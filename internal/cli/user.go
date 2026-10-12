package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/logging"
	"github.com/hostrepute/reputecast/internal/store"
)

func newUserCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage accounts from the server (works without a sending provider)",
	}
	var email, reason, password string
	withStore := func(cmd *cobra.Command, fn func(q *store.Queries, al *audit.Logger, u store.User) error) error {
		cfg, log, err := o.load(cmd)
		if err != nil {
			return err
		}
		pool, err := db.Open(cmd.Context(), cfg.Database.URL, 2)
		if err != nil {
			return err
		}
		defer pool.Close()
		q := store.New(pool)
		e, err := normalizeEmail(email)
		if err != nil {
			return fmt.Errorf("--email: %w", err)
		}
		u, err := q.GetUserByEmail(cmd.Context(), e)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("no account for %s", e)
		} else if err != nil {
			return err
		}
		_ = logging.ParseLevel
		return fn(q, audit.New(q, log), u)
	}
	cli := &audit.Actor{Type: authz.ActorSystem, Name: "CLI (server)"}

	reset := &cobra.Command{
		Use:   "reset-password",
		Short: "Print a one-time password reset link (works without a sending provider)",
		Long: "Print a one-time link that lets the person set a new password. It works once and expires in\n" +
			"15 minutes; their current password keeps working until they use it. With --password the\n" +
			"password is set directly instead and every session is signed out.",
		Example: "  reputecast user reset-password --email aisha@acme.io",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withStore(cmd, func(q *store.Queries, al *audit.Logger, u store.User) error {
				ctx := cmd.Context()
				out := cmd.OutOrStdout()
				if password == "" {
					if u.Status != "active" {
						return fmt.Errorf("%s is deactivated; reactivate the account first", u.Email)
					}
					token, expires, err := accounts.CreateResetToken(ctx, q, u.ID, "cli", time.Now())
					if err != nil {
						return err
					}
					base := cfgPublicURL(cmd, o)
					if st, err := q.GetInstanceSettings(ctx); err == nil && st.PublicUrl != "" {
						base = st.PublicUrl
					}
					_ = al.Log(ctx, audit.Event{Action: "user.password_reset_link", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: cli, Reason: "reset link from the server CLI"})
					fmt.Fprintf(out, "One-time password reset link for %s (works once, until %s):\n\n  %s\n\n", u.Email, expires.Format("15:04 MST"), accounts.ResetLink(base, token))
					fmt.Fprintln(out, "Send it to them privately. Their current password keeps working until they use it.")
					return nil
				}
				cfg, _, err := o.load(cmd)
				if err != nil {
					return err
				}
				if p := authn.PasswordProblem(password, cfg.Auth.PasswordMinLength); p != "" {
					return usageError{errors.New(p)}
				}
				hash, err := authn.HashPassword(password)
				if err != nil {
					return err
				}
				if err := q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: u.ID, PasswordHash: &hash, MustChangePassword: false}); err != nil {
					return err
				}
				_, _ = q.RevokeAllUserSessions(ctx, u.ID)
				_ = q.InvalidatePasswordResets(ctx, u.ID)
				_ = al.Log(ctx, audit.Event{Action: "user.password_reset", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: cli, Reason: "password set from the server CLI"})
				fmt.Fprintf(out, "Password set for %s. Every session was signed out.\n", u.Email)
				return nil
			})
		},
	}
	reset.Flags().StringVar(&email, "email", "", "account email")
	reset.Flags().StringVar(&password, "password", "", "set this password directly instead of printing a reset link")

	reset2fa := &cobra.Command{
		Use:     "reset-2fa",
		Short:   "Remove the authenticator app (passkeys stay); requires a reason, which is logged",
		Example: "  reputecast user reset-2fa --email aisha@acme.io --reason \"lost phone, verified by video call\"",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(reason) == "" {
				return usageError{errors.New("--reason is required; it's written to the audit log")}
			}
			return withStore(cmd, func(q *store.Queries, al *audit.Logger, u store.User) error {
				ctx := cmd.Context()
				if err := q.ResetUserTwoFactor(ctx, u.ID); err != nil {
					return err
				}
				_ = q.DeleteRecoveryCodes(ctx, u.ID)
				grace := time.Now().Add(72 * time.Hour)
				_ = q.SetUserTwoFAGrace(ctx, store.SetUserTwoFAGraceParams{ID: u.ID, TwofaGraceUntil: &grace})
				_, _ = q.RevokeAllUserSessions(ctx, u.ID)
				_ = al.Log(ctx, audit.Event{Action: "user.twofa_reset", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: cli, Reason: reason})
				fmt.Fprintf(cmd.OutOrStdout(), "Two-factor reset for %s. They have 72 hours to set it up again.\n", u.Email)
				return nil
			})
		},
	}
	reset2fa.Flags().StringVar(&email, "email", "", "account email")
	reset2fa.Flags().StringVar(&reason, "reason", "", "why (written to the audit log)")

	makeAdmin := &cobra.Command{
		Use:   "make-admin",
		Short: "Make an account an instance admin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withStore(cmd, func(q *store.Queries, al *audit.Logger, u store.User) error {
				ctx := cmd.Context()
				if err := q.SetUserInstanceAdmin(ctx, store.SetUserInstanceAdminParams{ID: u.ID, IsInstanceAdmin: true}); err != nil {
					return err
				}
				_ = al.Log(ctx, audit.Event{Action: "user.instance_admin_grant", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: cli})
				fmt.Fprintf(cmd.OutOrStdout(), "%s is now an instance admin.\n", u.Email)
				return nil
			})
		},
	}
	makeAdmin.Flags().StringVar(&email, "email", "", "account email")

	unlock := &cobra.Command{
		Use:   "unlock",
		Short: "Clear a sign-in lockout",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withStore(cmd, func(q *store.Queries, al *audit.Logger, u store.User) error {
				ctx := cmd.Context()
				if err := q.RecordSignInSuccess(ctx, u.ID); err != nil {
					return err
				}
				_ = al.Log(ctx, audit.Event{Action: "user.unlock", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: cli})
				fmt.Fprintf(cmd.OutOrStdout(), "%s can sign in again.\n", u.Email)
				return nil
			})
		},
	}
	unlock.Flags().StringVar(&email, "email", "", "account email")

	cmd.AddCommand(reset, reset2fa, makeAdmin, unlock)
	return cmd
}

// cfgPublicURL is the configured public URL (used before instance settings exist).
func cfgPublicURL(cmd *cobra.Command, o *rootOpts) string {
	cfg, _, err := o.load(cmd)
	if err != nil {
		return ""
	}
	return cfg.HTTP.PublicURL
}
