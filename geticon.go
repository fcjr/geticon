package geticon

import "unsafe"

func cToGoSlice(ptr unsafe.Pointer, len int) []byte {
	// support arbitrary len slices
	// see https://github.com/crawshaw/sqlite/issues/45
	slice := struct {
		data unsafe.Pointer
		len  int
		cap  int
	}{
		data: ptr,
		len:  int(len),
		cap:  int(len),
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
