package geticon

import "unsafe"

const pngHeader = "\x89PNG\r\n\x1a\n"

func isPNG(b []byte) bool {
	return len(b) >= 8 && string(b[:8]) == pngHeader
}

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
