package cli

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

type installInput struct {
	Email    string
	Name     string
	Password string
}

type installResult struct {
	Email        string
	TempPassword string // empty when the password came from the environment
}

func newInstallCmd(o *rootOpts) *cobra.Command {
	var in installInput
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Migrate the database and create the first Owner",
		Long: "Apply migrations and create the first Owner (an instance admin with break-glass sign-in).\n" +
			"Without --password a temporary password is printed; it must be replaced at first sign-in,\n" +
			"where the Owner also adds a passkey or authenticator, saves recovery codes and creates the first workspace.",
		Example: "  reputecast install --email linh@acme.io --name \"Linh Nguyen\"",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			if err := db.MigrateUp(cmd.Context(), pool, log); err != nil {
				return err
			}
			res, err := installIfEmpty(cmd.Context(), store.New(pool), cfg, in)
			if err != nil {
				return err
			}
			if res == nil {
				return errors.New("ReputeCast is already installed: users exist. Use `reputecast user` to manage accounts")
			}
			printInstall(cmd.OutOrStdout(), cfg, res)
			return nil
		},
	}
	cmd.Flags().StringVar(&in.Email, "email", "", "Owner email (or REPUTECAST_ADMIN_EMAIL)")
	cmd.Flags().StringVar(&in.Name, "name", "", "Owner name (or REPUTECAST_ADMIN_NAME)")
	cmd.Flags().StringVar(&in.Password, "password", "", "Owner password (or REPUTECAST_ADMIN_PASSWORD); omit to print a temporary one")
	return cmd
}

// installIfEmpty creates the first Owner when there are no users yet. It
// returns nil when users already exist.
func installIfEmpty(ctx context.Context, q *store.Queries, cfg *config.Config, in installInput) (*installResult, error) {
	n, err := q.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, nil
	}
	if in.Email == "" {
		in.Email = os.Getenv("REPUTECAST_ADMIN_EMAIL")
	}
	if in.Name == "" {
		in.Name = os.Getenv("REPUTECAST_ADMIN_NAME")
	}
	if in.Password == "" {
		in.Password = os.Getenv("REPUTECAST_ADMIN_PASSWORD")
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, fmt.Errorf("owner email: %w (pass --email or set REPUTECAST_ADMIN_EMAIL)", err)
	}
	res := &installResult{Email: email}
	mustChange := false
	if in.Password == "" {
		in.Password = tempPassword()
		res.TempPassword = in.Password
		mustChange = true
	} else if p := authn.PasswordProblem(in.Password, cfg.Auth.PasswordMinLength); p != "" {
		return nil, errors.New(p)
	}
	hash, err := authn.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	_, err = q.CreateUser(ctx, store.CreateUserParams{
		ID:                 uuid.Must(uuid.NewV7()),
		Email:              email,
		Name:               strings.TrimSpace(in.Name),
		Timezone:           "UTC",
		PasswordHash:       &hash,
		PasswordSetAt:      &now,
		MustChangePassword: mustChange,
		IsInstanceAdmin:    true,
		BreakGlass:         true,
		AvatarColor:        "info",
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func printInstall(w io.Writer, cfg *config.Config, res *installResult) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ReputeCast is installed.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Sign in at  %s\n", cfg.HTTP.PublicURL)
	fmt.Fprintf(w, "  Email       %s\n", res.Email)
	if res.TempPassword != "" {
		fmt.Fprintf(w, "  Password    %s   (temporary: you'll replace it at first sign-in)\n", res.TempPassword)
	} else {
		fmt.Fprintln(w, "  Password    the one you set")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  At first sign-in you'll add a passkey or authenticator app, save recovery codes,")
	fmt.Fprintln(w, "  confirm the public URL and create the first workspace.")
	fmt.Fprintln(w)
}

func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", errors.New("missing")
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || !strings.Contains(s, "@") {
		return "", fmt.Errorf("%q is not a valid email address", s)
	}
	return s, nil
}

// tempPassword returns four groups of four characters (about 95 bits).
func tempPassword() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	var b strings.Builder
	for i, c := range buf {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(alphabet[int(c)%len(alphabet)])
	}
	return b.String()
}
