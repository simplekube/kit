package pointer

import "time"

func Int(i int) *int {
	o := i
	return &o
}

func Int32(i int32) *int32 {
	o := i
	return &o
}

func Int64(i int64) *int64 {
	o := i
	return &o
}

func Bool(b bool) *bool {
	o := b
	return &o
}

func Duration(t time.Duration) *time.Duration {
	o := t
	return &o
}
