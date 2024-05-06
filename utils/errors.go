package utils

import (
	"fmt"
)

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Not found: %s", e.Message)
	} else {
		return fmt.Sprintf("Not found")
	}
}

type OutOfSpaceError struct {
	Allowed int64
	Current int64
	Units   string
	Reason  string
}

func (e *OutOfSpaceError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("Out of space: %d / %d %s (%s)", e.Current, e.Allowed, e.Units, e.Reason)
	} else {
		return fmt.Sprintf("Out of space: %d / %d %s", e.Current, e.Allowed, e.Units)
	}
}
