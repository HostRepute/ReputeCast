package db

import "strings"

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLike makes s match itself in a LIKE pattern: a search for "50%"
// or "first_name" finds those characters, not any text.
func EscapeLike(s string) string { return likeEscaper.Replace(s) }
