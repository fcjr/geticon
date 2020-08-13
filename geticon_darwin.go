package geticon

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreFoundation -framework Foundation -framework AppKit

#import <stdlib.h>
#import <Foundation/Foundation.h>
#import <AppKit/NSImage.h>
#import <AppKit/NSRunningApplication.h>

int getIcon(pid_t pid, void **img, int *imglen) {
	NSRunningApplication * app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
	if (app == nil) {
		return 1;
	}
	NSImage *appIcon = [app icon];
	if (appIcon == nil) {
		[app release];
		return 1;
	}
	NSData *tiffData = [appIcon TIFFRepresentation];
	[app release];
	[appIcon release];

	*imglen = (int) [tiffData length];
	*img = malloc(*imglen);
	if (*img == NULL) {
		[tiffData release];
		return 1;
	}
	memcpy(*img, [tiffData bytes], *imglen);
	[tiffData release];
	return 0;
}
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"unsafe"

	"golang.org/x/image/tiff"
)

// FromPid returns the app icon of the app currently running
// on the given pid, if it has one.
// This function will fail if the given PID does not have an
// icon associated with it.
func FromPid(pid uint32) (image.Image, error) {
	var imgLen C.int
	var imgPntr unsafe.Pointer
	errCode := C.getIcon(C.pid_t(pid), &imgPntr, &imgLen)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to gather icon")
	}
	defer C.free(imgPntr)

	// support arbitrary len slices
	// see https://github.com/crawshaw/sqlite/issues/45
	slice := struct {
		data unsafe.Pointer
		len  int
		cap  int
	}{
		data: imgPntr,
		len:  int(imgLen),
		cap:  int(imgLen),
	}
	imgData := *(*[]byte)(unsafe.Pointer(&slice))

	img, err := tiff.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}
	return img, nil
}
