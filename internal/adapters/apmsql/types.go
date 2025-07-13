package apmsql

import "time"

// QueryInfo holds information about a single SQL query collected by the probe.
// It is populated by the wrapped database driver and later analysed by the
// HTTP-layer N+1 detector.
type QueryInfo struct {
	Query    string
	Duration time.Duration
}
