package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                   = windows.NewLazyDLL("kernel32.dll")
	queryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	loadLibraryEx              = kernel32.NewProc("LoadLibraryExW")
	enumResourceNamesA         = kernel32.NewProc("EnumResourceNamesA")
)

const (
	// https://docs.microsoft.com/en-us/windows/win32/menurc/resource-types
	RT_ICON       = 3
	RT_GROUP_ICON = RT_ICON + 11

	// https://docs.microsoft.com/en-us/windows/win32/api/libloaderapi/nf-libloaderapi-loadlibraryexa
	LOAD_LIBRARY_AS_IMAGE_RESOURCE = 0x00000020
	LOAD_LIBRARY_AS_DATAFILE       = 0x00000002
)

type (
	LPEnumFunc func(hModule uintptr, lpType string, lpName string, wLanguage uint16, lParam *int) bool
)

func QueryFullProcessImageName(pid uint32, flags uint32) (s string, err error) {
	if err := queryFullProcessImageNameW.Find(); err != nil {
		return "", err
	}

	c, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(c)

	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(windows.MAX_LONG_PATH)
	ret, _, err := queryFullProcessImageNameW.Call(
		uintptr(c),
		uintptr(flags),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return "", err
	}
	return windows.UTF16ToString(buf[:]), nil

}

func LoadLibraryEx(name string, flags uint32) (windows.Handle, error) {
	if err := loadLibraryEx.Find(); err != nil {
		return 0, err
	}

	var namePtr *uint16
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r1, _, e1 := loadLibraryEx.Call(
		uintptr(unsafe.Pointer(namePtr)),
		0,
		uintptr(flags),
	)
	handle := windows.Handle(r1)
	if handle == 0 {
		if e1 != nil {
			fmt.Println(e1)
			return 0, e1
		}
	}
	return handle, nil

}

func MakeIntResource(id uint16) *uint16 {
	return (*uint16)(unsafe.Pointer(uintptr(id)))
}

func EnumResourceNamesA(hModule windows.Handle, lpcStr *uint16, lpEnumFunc LPEnumFunc, lParam *int32) error {
	if err := enumResourceNamesA.Find(); err != nil {
		return err
	}
	r1, _, e1 := enumResourceNamesA.Call(
		uintptr(hModule),
		uintptr(unsafe.Pointer(&lpcStr)),
		uintptr(unsafe.Pointer(&lpEnumFunc)),
		uintptr(unsafe.Pointer(&lParam)),
	)
	if r1 != 0 {
		return windows.Errno(r1)
	}
	return e1
}
