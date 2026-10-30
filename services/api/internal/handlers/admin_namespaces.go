package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// withProjectNamespaces appends every project's own namespace to base, so
// admin views also cover workloads outside the shared legacy namespace.
func withProjectNamespaces(ctx context.Context, pool *pgxpool.Pool, base ...string) []string {
	out := append([]string{}, base...)
	if pool == nil {
		return out
	}
	seen := make(map[string]bool, len(base))
	for _, ns := range base {
		seen[ns] = true
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT namespace FROM projects WHERE namespace <> ''`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var ns string
		if rows.Scan(&ns) == nil && !seen[ns] {
			seen[ns] = true
			out = append(out, ns)
		}
	}
	return out
}
