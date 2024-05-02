package kland

import (
	"fmt"
)

type NotFoundError struct {
	Lookup string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("Can't find: %s", e.Lookup)
}
