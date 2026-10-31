package polls

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure keeps poll answers and survey responses counted, unlinked from the
// contact, and drops what the person wrote (design expansion X11; review
// 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "polls",
		Tables: []erasure.Table{
			{Name: "poll_answers", Action: "clear", Columns: []string{"contact_id", "message_id", "comment"},
				Kept: map[string]string{"test_email": "a staff test address, not a contact's"}},
			{Name: "survey_responses", Action: "clear", Columns: []string{"contact_id", "message_id", "answers"},
				Kept: map[string]string{"test_email": "a staff test address, not a contact's"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE poll_answers SET contact_id = NULL, message_id = NULL, comment = '' WHERE contact_id = ANY ($1)`,
				`UPDATE survey_responses SET contact_id = NULL, message_id = NULL,
				    answers = (SELECT coalesce(jsonb_object_agg(e.k, e.v), '{}'::jsonb) FROM jsonb_each(survey_responses.answers) AS e(k, v) WHERE jsonb_typeof(e.v) <> 'string')
				 WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "pollAnswers", SQL: `SELECT p.question, a.answer, a.comment, a.answered_at FROM poll_answers a JOIN polls p ON p.id = a.poll_id WHERE a.contact_id = $1 ORDER BY a.answered_at`},
			{Section: "surveyResponses", SQL: `SELECT s.name AS survey, r.answers, r.started_at, r.completed_at FROM survey_responses r JOIN surveys s ON s.id = r.survey_id WHERE r.contact_id = $1 ORDER BY r.started_at`},
		},
	}
}
