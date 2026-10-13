package segments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SQL segments: the person writes a SELECT over the segment views that
// returns contact ids in a column named id. It runs as reputecast_segments
// (a role that can read nothing but the views granted for this one query)
// through reputecast_sql.segment_ids, a SECURITY DEFINER function owned by
// that role, inside a READ ONLY transaction with a statement timeout, and
// everything is rolled back. See migrations 00008 and 00151.

// MaxSQLLength bounds user-written SQL.
const MaxSQLLength = 20000

// Errors returned for user-written SQL.
var (
	// ErrSQLDisabled means the database has no reputecast_segments role or
	// runner function (managed Postgres without CREATEROLE).
	ErrSQLDisabled = errors.New("SQL segments are turned off on this database. An admin can enable them by running: CREATE ROLE reputecast_segments NOLOGIN; GRANT reputecast_segments TO <app user>; then reputecast migrate up")
	// ErrTimeout means the query hit the statement timeout.
	ErrTimeout = errors.New("count timed out")
)

// SQLError explains why user-written SQL was refused or failed, in words
// the person can act on.
type SQLError struct {
	Message  string
	Position int // 1-based character position in the user's SQL, 0 when unknown
}

func (e *SQLError) Error() string { return e.Message }

// CheckSQL validates the shape of user-written SQL before it reaches the
// database: one statement, starting with SELECT or WITH. It returns the
// statement without a trailing semicolon.
func CheckSQL(src string) (string, error) {
	s := strings.TrimSpace(src)
	if s == "" {
		return "", &SQLError{Message: "Write a SELECT that returns contact ids, for example: SELECT c.id FROM contacts c WHERE c.tier = 'active'"}
	}
	if len(s) > MaxSQLLength {
		return "", &SQLError{Message: fmt.Sprintf("SQL segments are limited to %d characters.", MaxSQLLength)}
	}
	toks, err := scan(s)
	if err != nil {
		return "", err
	}
	// Drop trailing semicolons; any other semicolon means a second statement.
	for len(toks) > 0 && toks[len(toks)-1].kind == tokSemicolon {
		s = strings.TrimSpace(s[:toks[len(toks)-1].pos])
		toks = toks[:len(toks)-1]
	}
	for _, t := range toks {
		if t.kind == tokSemicolon {
			return "", &SQLError{Message: "Only one statement is allowed: remove the semicolon and anything after it.", Position: t.pos + 1}
		}
	}
	first := ""
	for _, t := range toks {
		if t.kind == tokWord {
			first = strings.ToLower(t.text)
			break
		}
		if t.kind != tokPunct || t.text != "(" {
			break
		}
	}
	if first != "select" && first != "with" {
		return "", &SQLError{Message: "A SQL segment must be a SELECT (or WITH … SELECT) that returns contact ids."}
	}
	return s, nil
}

type tokKind int

const (
	tokWord tokKind = iota
	tokPunct
	tokSemicolon
	tokString
)

type token struct {
	kind tokKind
	text string
	pos  int
}

// scan is a small SQL lexer that understands comments, quoted strings,
// quoted identifiers and dollar quoting, so a semicolon inside them isn't
// mistaken for a statement separator.
func scan(s string) ([]token, error) {
	var out []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			depth, start := 0, i
			for i < len(s) {
				if s[i] == '/' && i+1 < len(s) && s[i+1] == '*' {
					depth++
					i += 2
					continue
				}
				if s[i] == '*' && i+1 < len(s) && s[i+1] == '/' {
					depth--
					i += 2
					if depth == 0 {
						break
					}
					continue
				}
				i++
			}
			if depth != 0 {
				return nil, &SQLError{Message: "A /* comment */ is never closed.", Position: start + 1}
			}
		case c == '\'' || ((c == 'e' || c == 'E') && i+1 < len(s) && s[i+1] == '\''):
			start := i
			escapes := c != '\''
			if escapes {
				i++
			}
			i++
			closed := false
			for i < len(s) {
				if escapes && s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, &SQLError{Message: "A quoted string is never closed.", Position: start + 1}
			}
			out = append(out, token{tokString, s[start:i], start})
		case c == '"':
			start := i
			i++
			closed := false
			for i < len(s) {
				if s[i] == '"' {
					if i+1 < len(s) && s[i+1] == '"' {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, &SQLError{Message: "A quoted identifier is never closed.", Position: start + 1}
			}
			out = append(out, token{tokWord, s[start:i], start})
		case c == '$':
			start := i
			j := i + 1
			for j < len(s) && (s[j] == '_' || isAlnum(s[j])) {
				j++
			}
			if j < len(s) && s[j] == '$' && (j == i+1 || !isDigit(s[i+1])) {
				tag := s[i : j+1]
				end := strings.Index(s[j+1:], tag)
				if end < 0 {
					return nil, &SQLError{Message: "A dollar-quoted string is never closed.", Position: start + 1}
				}
				i = j + 1 + end + len(tag)
				out = append(out, token{tokString, s[start:i], start})
				continue
			}
			i = j
			out = append(out, token{tokPunct, s[start:i], start})
		case c == ';':
			out = append(out, token{tokSemicolon, ";", i})
			i++
		case c == '_' || isAlnum(c) || c >= 0x80:
			start := i
			for i < len(s) && (s[i] == '_' || s[i] == '$' || isAlnum(s[i]) || s[i] >= 0x80) {
				i++
			}
			out = append(out, token{tokWord, s[start:i], start})
		case unicode.IsSpace(rune(c)):
			i++
		default:
			out = append(out, token{tokPunct, string(c), i})
			i++
		}
	}
	return out, nil
}

func isAlnum(c byte) bool { return isDigit(c) || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// SQLRunner executes user-written SQL in the restricted sandbox.
type SQLRunner struct {
	Pool    *pgxpool.Pool
	Timeout time.Duration

	once      sync.Once
	available bool
}

// Available reports whether the database has the runner function.
func (r *SQLRunner) Available(ctx context.Context) bool {
	r.once.Do(func() {
		var ok bool
		if err := r.Pool.QueryRow(ctx, `SELECT to_regprocedure('reputecast_sql.segment_ids(text)') IS NOT NULL`).Scan(&ok); err == nil {
			r.available = ok
		}
	})
	return r.available
}

func (r *SQLRunner) timeout() time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return 10 * time.Second
}

// Run executes fn in a READ ONLY transaction where the segment views exist
// for workspace ws and reputecast_sql.segment_ids can read them. Everything
// is rolled back. fn should select from reputecast_sql.segment_ids($n).
func (r *SQLRunner) Run(ctx context.Context, ws uuid.UUID, fn func(pgx.Tx) error) error {
	if !r.Available(ctx) {
		return ErrSQLDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout()+5*time.Second)
	defer cancel()
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx)) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", r.timeout().Milliseconds())); err != nil {
		return err
	}
	for _, stmt := range tempViews(ws) {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("segment views: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, "GRANT SELECT ON pg_temp.contacts, pg_temp.list_members, pg_temp.events, pg_temp.lists TO reputecast_segments"); err != nil {
		return fmt.Errorf("segment views grant: %w", err)
	}
	if _, err := tx.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return err
	}
	return mapSQLError(fn(tx))
}

// mapSQLError turns Postgres errors from user SQL into explanations.
func mapSQLError(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrTimeout
		}
		return err
	}
	msg := pg.Message
	switch pg.Code {
	case "57014":
		return ErrTimeout
	case "42501":
		return &SQLError{Message: "Your SQL can only read the contacts, list_members, events and lists views (" + msg + ")."}
	case "42804":
		if strings.Contains(msg, "structure of query") {
			return &SQLError{Message: "The query must return the contact id (a uuid) in a column named id, for example SELECT c.id FROM contacts c."}
		}
	case "42703":
		if strings.Contains(msg, `"id"`) {
			return &SQLError{Message: "The query must return the contact id in a column named id, for example SELECT c.id FROM contacts c."}
		}
	case "25006":
		return &SQLError{Message: "SQL segments are read-only: they can't change data."}
	}
	if strings.Contains(msg, "security-definer") {
		return &SQLError{Message: "SQL segments can't change roles or session settings."}
	}
	// Errors inside the runner point into its wrapper query; translate the
	// position back to the person's own SQL.
	pos := 0
	if pg.InternalPosition > 0 && strings.HasPrefix(pg.InternalQuery, runnerPrefix) {
		pos = int(pg.InternalPosition) - len(runnerPrefix)
	}
	return &SQLError{Message: capitalize(msg) + ".", Position: max(pos, 0)}
}

// runnerPrefix is how reputecast_sql.segment_ids wraps the person's SQL.
const runnerPrefix = "SELECT s.id::uuid FROM ("

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
