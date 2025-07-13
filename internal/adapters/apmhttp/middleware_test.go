package apmhttp

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fllarpy/apm-probe/internal/adapters/apmsql"
	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/fllarpy/apm-probe/pkg/config"
	_ "github.com/mattn/go-sqlite3" // Import for side effects
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB sets up an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	// 1. The `go-sqlite3` driver registers itself under the name "sqlite3".
	// We need to get a reference to it.
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	realDriver := db.Driver()
	err = db.Close()
	require.NoError(t, err)

	// 2. Register our wrapped driver.
	// We use a unique name for each test to avoid panics from re-registering.
	driverName := fmt.Sprintf("sqlite3-apm-%s", t.Name())
	apmsql.Register(driverName, realDriver)

	// 3. Open a connection using the wrapped driver.
	db, err = sql.Open(driverName, ":memory:")
	require.NoError(t, err, "Failed to open in-memory DB")

	// 3. Create a schema and seed data.
	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
		INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie');
	`)
	require.NoError(t, err, "Failed to create schema and seed data")

	return db
}

func TestMiddleware_NPlusOneDetection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := inmemory.NewStore()
	cfg := &config.Config{
		NPlusOneThreshold: 3, // Set threshold to 3 for this test
	}

	// This handler simulates an N+1 problem.
	nPlusOneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// It iterates and fetches users one by one in a loop.
		for i := 1; i <= 3; i++ {
			var name string
			// Pass the request context to the query.
			err := db.QueryRowContext(r.Context(), "SELECT name FROM users WHERE id = ?", i).Scan(&name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	// This handler is efficient.
	efficientHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// It fetches all users in a single query.
		rows, err := db.QueryContext(r.Context(), "SELECT name FROM users")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows.Close()
		w.WriteHeader(http.StatusOK)
	})

	// Create a test server with our middleware.
	mux := http.NewServeMux()
	mux.Handle("/n-plus-one", Middleware(cfg, store, nPlusOneHandler))
	mux.Handle("/efficient", Middleware(cfg, store, efficientHandler))
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// --- Act ---

	// Call the problematic endpoint.
	_, err := http.Get(testServer.URL + "/n-plus-one")
	require.NoError(t, err)

	// Call the efficient endpoint.
	_, err = http.Get(testServer.URL + "/efficient")
	require.NoError(t, err)

	// --- Assert ---

	snapshot := store.GetSnapshot()

	// 1. Check for N+1 events.
	assert.Len(t, snapshot.NPlusOneEvents, 1, "Expected exactly one N+1 event to be recorded")
	if len(snapshot.NPlusOneEvents) == 1 {
		event := snapshot.NPlusOneEvents[0]
		assert.Equal(t, "/n-plus-one", event.Path)
		assert.Equal(t, "SELECT name FROM users WHERE id = ?", event.Query)
		assert.Equal(t, 3, event.Count, "Expected the query to be repeated 3 times")
	}

	// 2. Ensure no N+1 events were recorded for the efficient endpoint.
	// We do this by checking the total count. If it's 1, it must be from the n-plus-one endpoint.
}
