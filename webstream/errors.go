package webstream

import (
	"fmt"
)

type RoomNameError struct {
	Regex string
}

func (e *RoomNameError) Error() string {
	return fmt.Sprintf("Room name has invalid characters! Regex: %s", e.Regex)
}

type RoomLimitError struct {
	Limit int
}

func (e *RoomLimitError) Error() string {
	return fmt.Sprintf("Room limit reached (%d), no new rooms can be created", e.Limit)
}

type ActiveRoomLimitError struct {
	Limit int
}

func (e *ActiveRoomLimitError) Error() string {
	return fmt.Sprintf("Active room limit reached (%d), must wait for another room to idle", e.Limit)
}

type OverCapacityError struct {
	Capacity int
}

func (e *OverCapacityError) Error() string {
	return fmt.Sprintf("data overflows capacity: %d", e.Capacity)
}
