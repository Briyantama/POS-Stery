package db

import "strings"

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLike escapes %, _ and \ so user input matches literally inside an ILIKE pattern.
func EscapeLike(s string) string { return likeEscaper.Replace(s) }
