package sso

import "context"

// MailApp returns the client ID, secret and (Microsoft) tenant of the
// instance's Google or Microsoft sign-in app (Instance admin ›
// Authentication) for placement tests' seed mailboxes, which connect with
// the same app. ok is false until both the ID and the secret are set; the
// app doesn't have to be turned on for signing in to ReputeCast.
func (s *Service) MailApp(ctx context.Context, kind string) (clientID, secret, tenant string, ok bool, err error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return "", "", "", false, err
	}
	c := SocialConfigs(st)[kind]
	sec := s.socialSecrets(st)[kind]
	return c.ClientID, sec, c.Tenant, c.ClientID != "" && sec != "", nil
}
