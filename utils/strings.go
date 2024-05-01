package utils

import (
	"strings"
)

func StringToBool(value *string) bool {
	var v string
	if value != nil {
		v = strings.ToLower(strings.TrimSpace(*value))
	}
	return v != "" && v != "0" && v != "false" && v != "undefined" && v != "null"
}

func Unpointer[T any](value *T, def T) T {
	if value == nil {
		return def
	}
	return *value
}

func IsNilOrEmpty(value *string) bool {
	return value == nil || *value == ""
}

func StrGetOrDefault(value *string, def string) string {
	vp := Unpointer(value, def)
	if vp == "" {
		return def
	}
	return vp
}
