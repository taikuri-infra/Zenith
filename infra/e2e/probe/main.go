// infra/e2e/probe/main.go
// e2e-probe: the sample app the E2E suite deploys. It reads and writes one table
// through DATABASE_URL so the suite can prove data survives reboot, upgrade and restore.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		var names []string
		for _, kv := range os.Environ() {
			k := strings.SplitN(kv, "=", 2)[0]
			if strings.HasPrefix(k, "DATABASE_URL") || strings.HasPrefix(k, "REDIS_URL") || strings.HasPrefix(k, "MYSQL_URL") {
				names = append(names, k)
			}
		}
		json.NewEncoder(w).Encode(names)
	})
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("v")
		if err := withDB(r.Context(), func(db *sql.DB) error {
			if _, err := db.ExecContext(r.Context(), `CREATE TABLE IF NOT EXISTS e2e (id serial primary key, v text not null)`); err != nil {
				return err
			}
			_, err := db.ExecContext(r.Context(), `INSERT INTO e2e(v) VALUES ($1)`, v)
			return err
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprint(w, "written")
	})
	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		var out []string
		if err := withDB(r.Context(), func(db *sql.DB) error {
			rows, err := db.QueryContext(r.Context(), `SELECT v FROM e2e ORDER BY id`)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var v string
				if err := rows.Scan(&v); err != nil {
					return err
				}
				out = append(out, v)
			}
			return rows.Err()
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if out == nil {
			out = []string{}
		}
		json.NewEncoder(w).Encode(out)
	})
	log.Printf("e2e-probe listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func withDB(ctx context.Context, fn func(*sql.DB) error) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return fn(db)
}
