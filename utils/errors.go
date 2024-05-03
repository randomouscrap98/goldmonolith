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
