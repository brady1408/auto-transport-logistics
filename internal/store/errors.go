package store

import "errors"

// ErrConflict is returned when an optimistic locking conflict is detected
// (i.e., the record was modified by another user since it was last read).
var ErrConflict = errors.New("concurrent modification conflict")
