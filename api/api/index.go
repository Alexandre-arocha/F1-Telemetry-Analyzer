// Package handler is the Vercel entrypoint: the same Gin router as cmd/server, served as a serverless function.
package handler

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alexandre-arocha/f1-telemetry-analyzer/api/internal/db"
	"github.com/Alexandre-arocha/f1-telemetry-analyzer/api/internal/httpapi"
)

var (
	once    sync.Once
	router  http.Handler
	initErr error
)

// Handler serves every request; the pool is created once per warm instance (migrations run via cmd/server).
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
		if err != nil {
			initErr = err
			return
		}
		cfg.MaxConns = 3 // serverless: many instances, keep each one small
		pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
		if err != nil {
			initErr = err
			return
		}
		origin := os.Getenv("CORS_ORIGIN")
		if origin == "" {
			origin = "*"
		}
		router = httpapi.New(db.New(pool), origin)
	})
	if initErr != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	// vercel.json rewrites every route to this function and keeps the original path in ?p=
	q := r.URL.Query()
	p := q.Get("p")
	q.Del("p")
	r.URL.RawQuery = q.Encode()
	if p == "healthz" {
		r.URL.Path = "/healthz"
	} else {
		r.URL.Path = "/api/" + p
	}
	router.ServeHTTP(w, r)
}
