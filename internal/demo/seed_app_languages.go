package demo

import "context"

func init() { seedAppLanguages = seedAppLanguagesImpl }

// seedAppLanguagesImpl sets the app languages of the connect-and-reach
// sample (brief §4, "App languages"):
//   - the instance default is English with the ReputeCast standard formats;
//   - Marcus Chen reads English with the English (United States) region
//     format, so he sees "Tue, Oct 14, 2026 · 9:00 AM";
//   - everyone else follows the instance default (Linh tries Tiếng Việt
//     live, in Language & region);
//   - jo@contractor.dev's invitation went out in English.
//
// Nothing here is dated, and the audit log keeps the size of the board
// (41,208 entries), so Marcus's change isn't added to it.
func seedAppLanguagesImpl(ctx context.Context, s *Seeder) error {
	if _, err := s.Tx.Exec(ctx, `UPDATE instance_settings SET default_language = 'en', default_region_format = 'standard'`); err != nil {
		return err
	}
	// The team seeder adds the invitation once every seeder has run.
	s.After(func(ctx context.Context, s *Seeder) error {
		_, err := s.Tx.Exec(ctx, `UPDATE invitations SET language = 'en' WHERE workspace_id = $1 AND email = 'jo@contractor.dev'`, s.WS().ID)
		return err
	})
	if marcus, ok := s.Users["marcus@acme.io"]; ok {
		if _, err := s.Tx.Exec(ctx, `UPDATE users SET language = 'en', region_format = 'en-US' WHERE id = $1`, marcus.ID); err != nil {
			return err
		}
	}
	return nil
}
