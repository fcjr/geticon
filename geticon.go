package geticon

import "unsafe"

func cToGoSlice(cPtr unsafe.Pointer, cLen int) []byte {
	// support arbitrary len slices
	// see https://github.com/crawshaw/sqlite/issues/45
	slice := struct {
		data unsafe.Pointer
		len  int
		cap  int
	}{
		data: cPtr,
		len:  cLen,
		cap:  cLen,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
