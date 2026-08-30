package runtimeinfo

import "unsafe"

type Value struct {
	Enabled bool
	Count   int64
	Code    uint16
}

type Layout struct {
	Size, Alignment uintptr
	EnabledOffset   uintptr
	CountOffset     uintptr
	CodeOffset      uintptr
}

func ValueLayout() Layout {
	var value Value
	return Layout{
		Size:          unsafe.Sizeof(value),
		Alignment:     unsafe.Alignof(value),
		EnabledOffset: unsafe.Offsetof(value.Enabled),
		CountOffset:   unsafe.Offsetof(value.Count),
		CodeOffset:    unsafe.Offsetof(value.Code),
	}
}
