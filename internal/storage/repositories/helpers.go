package repositories

import (
	"database/sql"
	"time"
)

// formatTime renders a timestamp as RFC3339 in UTC for stable text storage.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// parseTime reverses formatTime, returning the zero time on malformed input.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}

	return t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// nullableInt maps an optional int pointer onto a SQL NULL when absent.
func nullableInt(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: int64(*p), Valid: true}
}
