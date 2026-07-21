package infra

import "errors"

// ErrNoRows is returned when a query does not find rows.
var ErrNoRows = errors.New("no rows in result set")
