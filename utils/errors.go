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

type InvalidName struct {
	Item  string
	Regex string
}

func (e *InvalidName) Error() string {
	msg := "Invalid characters"
	if e.Item != "" {
		msg += " in " + e.Item
	}
	if e.Regex != "" {
		msg += ". Regex: " + e.Regex
	}
	return msg
}

// An error that "is to be expected", meaning we can handle it somehow
type ExpectedError struct {
	Message string
}

func (e *ExpectedError) Error() string {
	return e.Message
}
