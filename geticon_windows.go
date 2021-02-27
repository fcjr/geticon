package geticon

import (
	"fmt"
	"image"

	"github.com/fcjr/geticon/internal/winapi"
	"golang.org/x/sys/windows"
)

// FromPid returns the app icon of the app currently running
// on the given pid, if it has one.
// This function will fail if the given PID does not have an
// icon associated with it.
func FromPid(pid uint32) (image.Image, error) {
	// get path from pid
	exePath, err := winapi.QueryFullProcessImageName(pid, 0)
	if err != nil {
		return nil, err
	}

	exeHandle, err := winapi.LoadLibraryEx(
		exePath,
		winapi.LOAD_LIBRARY_AS_IMAGE_RESOURCE&winapi.LOAD_LIBRARY_AS_DATAFILE,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(exeHandle)

	callback := func(hModule uintptr, lpType string, lpName string, wLanguage uint16, lParam *int) bool {
		fmt.Println("callback called")
		return false
	}

	var data int32
	err = winapi.EnumResourceNamesA(
		exeHandle,
		winapi.MakeIntResource(winapi.RT_GROUP_ICON),
		callback,
		&data,
	)
	if err != nil {
		fmt.Println("failed enum")
		return nil, err
	}

	return nil, fmt.Errorf("unimplemented")
}
