package geticon

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreFoundation -framework Foundation -framework AppKit

#import <stdlib.h>
#import <Foundation/Foundation.h>
#import <AppKit/NSImage.h>
#import <AppKit/NSRunningApplication.h>

int getIcon(void* imgBuf, pid_t pid) {
	NSRunningApplication * app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
	if (app == nil) {
		return -1;
	}
	NSImage *appIcon = [app icon];
	if (appIcon == nil) {
		return -1;
	}
	NSData *tiffData = [appIcon TIFFRepresentation];
	if ([tiffData length] > (1 << 28)){
		return -1;
	}
	memcpy(imgBuf, [tiffData bytes], [tiffData length]);
	return [tiffData length];
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
	// TODO don't allocate all this space
	var imgBuf [1 << 28]byte
	length := C.getIcon(unsafe.Pointer(&imgBuf[0]), C.pid_t(pid))
	if length < 0 {
		return nil, fmt.Errorf("failed to gather icon")
	}
	tmpData := imgBuf[0:length]
	img, err := tiff.Decode(bytes.NewReader(tmpData))
	if err != nil {
		return nil, err
	}
	return img, nil
}
