package forms

import "time"

// TokenAge opens a form token and returns how long ago it was issued, for
// forms whose own timing check isn't a signup form's (comments on posts
// need 3 seconds to 24 hours). ok is false for a token that is forged,
// for another form, or older than a day.
func (s *Service) TokenAge(formKey, token string, now time.Time) (time.Duration, bool) {
	return s.tokenAge(formKey, token, now)
}
