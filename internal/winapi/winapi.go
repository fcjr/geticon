package winapi

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                   = windows.NewLazyDLL("kernel32.dll")
	queryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	loadLibraryExW             = kernel32.NewProc("LoadLibraryExW")
	enumResourceNamesA         = kernel32.NewProc("EnumResourceNamesA")
	findResourceA              = kernel32.NewProc("FindResourceA")
	sizeofResource             = kernel32.NewProc("SizeofResource")
	loadResource               = kernel32.NewProc("LoadResource")
	lockResource               = kernel32.NewProc("LockResource")
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
	ENUMRESNAMEPROCA func(hModule windows.Handle, lpType, lpName, lParam uintptr) uintptr
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
	if err := loadLibraryExW.Find(); err != nil {
		return 0, err
	}

	var namePtr *uint16
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r1, _, e1 := loadLibraryExW.Call(
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

func MakeIntResource(id uintptr) *uint16 {
	return (*uint16)(unsafe.Pointer(id))
}

func IsIntResource(id uintptr) bool {
	return *(*uint16)(unsafe.Pointer(id))<<16 == 0
}

func EnumResourceNames(hModule windows.Handle, lpcStr *uint16, lpEnumFunc ENUMRESNAMEPROCA, lParam *int32) error {
	if err := enumResourceNamesA.Find(); err != nil {
		return err
	}
	r1, _, e1 := enumResourceNamesA.Call(
		uintptr(hModule),
		uintptr(unsafe.Pointer(lpcStr)),
		syscall.NewCallback(lpEnumFunc),
		uintptr(unsafe.Pointer(lParam)),
	)
	if r1 != 0 {
		return windows.Errno(r1)
	}
	return e1
}

func FindResource(hModule windows.Handle, lpName, lpType uintptr) (rInfo uintptr, err error) {
	if err := findResourceA.Find(); err != nil {
		return 0, err
	}
	r1, _, e1 := findResourceA.Call(
		uintptr(hModule),
		lpName,
		lpType,
	)
	if r1 == 0 {
		if e1 != nil {
			return 0, err
		} else {
			return 0, fmt.Errorf("couldn't find resource")
		}
	}
	return r1, nil
}

func SizeofResource(hModule windows.Handle, hResInfo uintptr) (uint32, error) {
	if err := sizeofResource.Find(); err != nil {
		return 0, err
	}
	r1, _, e1 := sizeofResource.Call(
		uintptr(hModule),
		hResInfo,
	)
	if e1 == windows.Errno(0) {
		return uint32(r1), nil
	}
	return 0, e1
}

func LoadResource(hModule windows.Handle, hResInfo uintptr) (windows.Handle, error) {
	if err := loadResource.Find(); err != nil {
		return 0, err
	}
	r1, _, e1 := loadResource.Call(
		uintptr(hModule),
		hResInfo,
	)
	if e1 == windows.Errno(0) {
		return windows.Handle(r1), nil
	}
	return 0, e1
}

func LockResource(hResData windows.Handle) (windows.Handle, error) {
	if err := lockResource.Find(); err != nil {
		return 0, err
	}

	r1, _, _ := lockResource.Call(
		uintptr(hResData),
		0,
		0)

	return windows.Handle(r1), nil
}
