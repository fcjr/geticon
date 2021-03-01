package geticon

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit

#import <stdlib.h>
#import <AppKit/NSImage.h>
#import <AppKit/NSRunningApplication.h>

int getIcon(NSImage *appIcon, void **img, int *imglen) {
	NSData *tiffData = [appIcon TIFFRepresentation];
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

int getIconFromPid(pid_t pid, void **img, int *imglen) {
	NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
	if (app == nil) {
		return 1;
	}
	NSImage *appIcon = [app icon];
	if (appIcon == nil) {
		[app release];
		return 1;
	}
	[app release];

	return getIcon(appIcon, img, imglen);
}

int getIconFromPath(char* path, void** img, int *imglen) {
	NSString *bundlePath = [NSString stringWithUTF8String:path];
	printf("%s\n", [bundlePath UTF8String]);
	NSImage *appIcon = [[NSWorkspace sharedWorkspace] iconForFile:bundlePath];
	if (appIcon == nil) {
		return 1;
	}

	return getIcon(appIcon, img, imglen);
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

// FromPath returns the app icon app at the specified path.
func FromPath(appPath string) (image.Image, error) {
	var imgLen C.int
	var imgPntr unsafe.Pointer

	cPath := C.CString(appPath)
	defer C.free(unsafe.Pointer(cPath))

	errCode := C.getIconFromPath(cPath, &imgPntr, &imgLen)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to gather icon")
	}
	defer C.free(imgPntr)

	imgData := cToGoSlice(imgPntr, int(imgLen))

	img, err := tiff.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// FromPid returns the app icon of the app currently running
// on the given pid.
func FromPid(pid uint32) (image.Image, error) {
	var imgLen C.int
	var imgPntr unsafe.Pointer
	errCode := C.getIconFromPid(C.pid_t(pid), &imgPntr, &imgLen)
	if errCode != 0 {
		return nil, fmt.Errorf("failed to gather icon")
	}
	defer C.free(imgPntr)

	imgData := cToGoSlice(imgPntr, int(imgLen))

	img, err := tiff.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}
	return img, nil
}
