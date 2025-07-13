package sql

import (
	"database/sql"

	"github.com/XSAM/otelsql"
)

func Open(driverName, dataSourceName string) (*sql.DB, error) {
	db, err := otelsql.Open(driverName, dataSourceName, otelsql.WithAttributes())
	if err != nil {
		return nil, err
	}

	return db, nil
}
