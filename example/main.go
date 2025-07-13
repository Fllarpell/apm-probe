package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"

	apm "github.com/fllarpy/apm-probe"
	httpinstrumentation "github.com/fllarpy/apm-probe/instrumentation/http"
	sqlinstrumentation "github.com/fllarpy/apm-probe/instrumentation/sql"
	"github.com/fllarpy/apm-probe/internal/ports/http_reporter"
)

func main() {
	ctx := context.Background()
	probe, store, err := apm.NewProbe(ctx, "hybrid-service")
	if err != nil {
		log.Fatalf("failed to initialize apm probe: %v", err)
	}
	defer probe.Shutdown(ctx)

	db, err := sqlinstrumentation.Open("sqlite3", "file:./apm-probe/example/test.db?cache=shared&mode=memory")
	if err != nil {
		log.Fatalf("failed to open instrumented db connection: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", helloHandler)
	mux.HandleFunc("/db", dbHandler(db))
	mux.HandleFunc("/error", erroringHandler)
	mux.HandleFunc("/db-error", dbErroringHandler(db))
	mux.HandleFunc("/slow", slowHandler)
	mux.HandleFunc("/n-plus-one", nPlusOneHandler(db))

	metricsHandler := http_reporter.NewHandler(store)
	mux.Handle("/debug/apm", metricsHandler)

	instrumentedHandler := httpinstrumentation.NewMiddleware(mux, "http-server")

	log.Printf("Starting server for service 'hybrid-service' on :8080")
	log.Println("Test endpoint: http://localhost:8080/")
	log.Println("DB test endpoint: http://localhost:8080/db")
	log.Println("Error test endpoint: http://localhost:8080/error")
	log.Println("DB Error test endpoint: http://localhost:8080/db-error")
	log.Println("Slow endpoint for profiling: http://localhost:8080/slow")
	log.Println("N+1 test endpoint: http://localhost:8080/n-plus-one")
	log.Println("Legacy metrics endpoint: http://localhost:8080/debug/apm")

	if err := http.ListenAndServe(":8080", instrumentedHandler); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(50 * time.Millisecond)
	fmt.Fprintln(w, "Hello, from the hybrid server!")
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(600 * time.Millisecond)
	fmt.Fprintln(w, "This was a slow request.")
}

func erroringHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, "This endpoint always returns an error.")
}

func dbHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		row := db.QueryRowContext(ctx, "SELECT 'John Doe' as name")
		var name string
		if err := row.Scan(&name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "User name from DB: %s\n", name)
	}
}

func dbErroringHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, err := db.ExecContext(ctx, "SELECT * FROM non_existent_table")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "This should not be reached.")
	}
}

func nPlusOneHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		for i := 0; i < 10; i++ {
			row := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", 1)
			var name string
			_ = row.Scan(&name)
		}
		fmt.Fprintln(w, "Executed 10 identical queries.")
	}
}
