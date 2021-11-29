package envutil

import (
	"os"
	"strings"
)

// string representations of boolean true & boolean false
var _falsy = "false"
var _truthy = "true"

func Disable(key string) {
	os.Setenv(key, _falsy)
}

func IsEnabled(key string, fallback bool) bool {
	var fb = _falsy
	if fallback {
		fb = _truthy
	}
	v := GetOrDefault(key, fb)
	if v != _truthy && v != _falsy {
		// set to false if value is not a boolean representation
		v = _falsy
	}
	return v == _truthy
}

func MayBeEnable(key string) {
	MayBeSet(key, _truthy)
}

func MayBeSet(key string, value string) string {
	existing, found := os.LookupEnv(key)
	existing = strings.TrimSpace(existing)

	// if not found or is not set already
	if !found || existing == "" {
		err := os.Setenv(key, value)
		if err != nil {
			panic(err)
		}
		return value
	}
	return existing
}

func LookupOrDefault(key string, fallback string) string {
	val, found := os.LookupEnv(key)
	if !found {
		return fallback
	}
	return strings.TrimSpace(val)
}

func GetOrDefault(key string, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
