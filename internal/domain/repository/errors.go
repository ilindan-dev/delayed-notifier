package repository

import "errors"

// ErrNotFound is returned when a record is not found in the database.
var ErrNotFound = errors.New("record not found")

// ErrDuplicateRecord is returned when an insert operation violates a UNIQUE constraint.
var ErrDuplicateRecord = errors.New("duplicate record")
