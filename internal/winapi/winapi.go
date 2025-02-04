//go:build windows
// +build windows

package winapi

import (
	"fmt"
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
	// RtIcon Hardware-dependent icon resource.
	// MAKEINTRESOURCE((ULONG_PTR)(RT_ICON) + 11)
	// https://docs.microsoft.com/en-us/windows/win32/menurc/resource-types
	RtIcon = 3
	// RtGroupIcon Hardware-independent icon resource.
	// MAKEINTRESOURCE((ULONG_PTR)(RT_ICON) + 11)
	// https://docs.microsoft.com/en-us/windows/win32/menurc/resource-types
	RtGroupIcon = RtIcon + 11

	// LoadLibraryAsImageResource 0x00000020
	// If this value is used, the system maps the file into the process's virtual address space as an image file.
	// However, the loader does not load the static imports or perform the other usual initialization steps.
	// Use this flag when you want to load a DLL only to extract messages or resources from it.
	// Unless the application depends on the file having the in-memory layout of an image, this value should be used
	// with either LOAD_LIBRARY_AS_DATAFILE_EXCLUSIVE or LOAD_LIBRARY_AS_DATAFILE.
	// For more information, see the Remarks section.
	//
	// Windows Server 2003 and Windows XP:  This value is not supported until Windows Vista.
	// https://docs.microsoft.com/en-us/windows/win32/api/libloaderapi/nf-libloaderapi-loadlibraryexa
	LoadLibraryAsImageResource = 0x00000020

	// LoadLibraryAsDatafile 0x00000002
	// If this value is used, the system maps the file into the calling process's virtual address space
	// as if it were a data file. Nothing is done to execute or prepare to execute the mapped file.
	// Therefore, you cannot call functions like GetModuleFileName, GetModuleHandle or GetProcAddress with
	// this DLL. Using this value causes writes to read-only memory to raise an access violation. Use this
	// flag when you want to load a DLL only to extract messages or resources from it.
	//
	// This value can be used with LOAD_LIBRARY_AS_IMAGE_RESOURCE. For more information, see Remarks.
	// https://docs.microsoft.com/en-us/windows/win32/api/libloaderapi/nf-libloaderapi-loadlibraryexa
	LoadLibraryAsDatafile = 0x00000002
)

type (
	// EnumResNameProcA is an application-defined callback function used with the EnumResourceNames
	// and EnumResourceNamesEx functions. It receives the type and name of a resource.
	// The ENUMRESNAMEPROC type defines a pointer to this callback function. EnumResNameProc is a
	// placeholder for the application-defined function name.
	EnumResNameProcA func(hModule windows.Handle, lpType, lpName, lParam uintptr) uintptr
)

// QueryFullProcessImageNameW retrieves the full name of the executable image for the specified process.
func QueryFullProcessImageNameW(pid uint32, flags uint32) (s string, err error) {
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

// LoadLibraryEx maps a specified executable module into the address space of the calling process.
// The executable module can be a .dll or an .exe file. The specified module may cause other modules
// to be mapped into the address space.
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
			return 0, e1
		}
		return 0, fmt.Errorf("library not found")
	}
	return handle, nil
}

// MakeIntResource converts an integer value to a resource type compatible with the resource-management functions.
// This macro is used in place of a string containing the name of the resource.
func MakeIntResource(i int) uintptr {
	return uintptr(uint16(i))
}

// EnumResourceNamesA enumerates resources of a specified type within a binary module. For Windows Vista and later,
// this is typically a language-neutral Portable Executable (LN file), and the enumeration will also include resources
// from the corresponding language-specific resource files (.mui files) that contain localizable language resources.
// It is also possible for hModule to specify an .mui file, in which case only that file is searched for resources.
func EnumResourceNamesA(hModule windows.Handle, lpcStr uintptr, lpEnumFunc EnumResNameProcA, lParam *int32) error {
	if err := enumResourceNamesA.Find(); err != nil {
		return err
	}
	r1, _, e1 := enumResourceNamesA.Call(
		uintptr(hModule),
		lpcStr,
		windows.NewCallback(lpEnumFunc),
		uintptr(unsafe.Pointer(lParam)),
	)
	if r1 != 0 {
		return windows.Errno(r1)
	}
	return e1
}

// FindResourceA etermines the location of a resource with the specified type and name in the specified module.
func FindResourceA(hModule windows.Handle, lpName, lpType uintptr) (rInfo uintptr, err error) {
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
		}
		return 0, fmt.Errorf("couldn't find resource")
	}
	return r1, nil
}

// SizeofResource retrieves the size, in bytes, of the specified resource.
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

// LoadResource retrieves a handle that can be used to obtain a pointer to the first byte
// of the specified resource in memory.
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

// LockResource retrieves a pointer to the specified resource in memory.
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
