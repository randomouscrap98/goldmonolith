package utils

import (
	"time"
)

const (
	ForeverDuration = "2000000h"
)

// A lot of things don't parse durations correctly; I need it.
type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
	bs := string(b)
	if bs == "never" || bs == "infinite" {
		bs = ForeverDuration
	}
	x, err := time.ParseDuration(bs)
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}
