package geticon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"unsafe"

	"github.com/fcjr/geticon/internal/winapi"
	"github.com/mat/besticon/ico"
	"golang.org/x/sys/windows"
)

// https://devblogs.microsoft.com/oldnewthing/20120720-00/?p=7083
type grpIconDir struct {
	IDReserved uint16
	IDType     uint16
	IDCount    uint16
}
type grpIconDirEntry struct {
	BWidth       byte
	BHeight      byte
	BColorCount  byte
	BReserved    byte
	WPlanes      uint16
	WBitCount    uint16
	DWBytesInRes uint32
	NID          uint16
}

type iconDirectoryEntry struct {
	BWidth        byte
	BHeight       byte
	BColorCount   byte
	BReserved     byte
	WPlanes       uint16
	WBitCount     uint16
	DWBytesinRes  uint32
	DWImageOffset uint32
}

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

	// get handle
	exeHandle, err := winapi.LoadLibraryEx(
		exePath,
		winapi.LOAD_LIBRARY_AS_IMAGE_RESOURCE|winapi.LOAD_LIBRARY_AS_DATAFILE,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(exeHandle)

	// var img image.Image
	icoBytes := []byte{}

	var innerErr error
	// enum rt_group_icons and grab the first one
	err = winapi.EnumResourceNames(
		exeHandle,
		winapi.MakeIntResource(winapi.RT_GROUP_ICON),
		func(hModule windows.Handle, lpType, lpName, lParam uintptr) uintptr {

			resPtr, _, err := getResource(hModule, lpType, lpName)
			if err != nil {
				innerErr = err
				return uintptr(0)
			}
			if resPtr == 0 {
				innerErr = fmt.Errorf("error no resource found")
				return uintptr(0)
			}

			iconHeader := *(*grpIconDir)(unsafe.Pointer(resPtr))
			// fmt.Println(iconHeader)

			// get entry info
			entriesPtr := resPtr + 6
			var entries []grpIconDirEntry
			for i := 0; i < int(iconHeader.IDCount); i++ {
				entry := *(*grpIconDirEntry)(unsafe.Pointer(entriesPtr + uintptr(i*14)))
				// fmt.Println(entry)
				entries = append(entries, entry)
			}

			// get icons
			imgs := [][]byte{}
			for _, entry := range entries {
				// fmt.Printf("%+v\n", entry)
				imgPntr, imgLen, err := getResource(
					hModule,
					winapi.RT_ICON,
					uintptr(entry.NID),
				)
				if err != nil {
					// fmt.Printf("failed to get resource %d, skipping...\n", entry.nId)
					// fmt.Println(err)
					innerErr = err
					return uintptr(0)
				}

				imgData := cToGoSlice(unsafe.Pointer(imgPntr), int(imgLen))
				// TODO figure out a better way to copy this
				imgCopy := make([]byte, imgLen)
				for i, b := range imgData {
					imgCopy[i] = b
				}

				imgs = append(imgs, imgCopy)
			}

			headerBytes, err := encodeToBytes(iconHeader)
			if err != nil {
				innerErr = err
				// fmt.Println(err)
				return uintptr(0)
			}
			icoBytes = append(icoBytes, headerBytes...)

			var offset uint32 = 6 + (uint32(iconHeader.IDCount) * 16)
			for i, entry := range entries {
				imgLen := uint32((len(imgs[i])))
				dirEntry := iconDirectoryEntry{
					BWidth:        entry.BWidth,
					BHeight:       entry.BHeight,
					BColorCount:   entry.BColorCount,
					BReserved:     entry.BReserved,
					WPlanes:       entry.WPlanes,
					WBitCount:     entry.WBitCount,
					DWBytesinRes:  imgLen,
					DWImageOffset: offset,
				}
				// fmt.Printf("%+v\n", dirEntry)

				entryBytes, err := encodeToBytes(dirEntry)
				if err != nil {
					innerErr = err
					return uintptr(0)
				}
				icoBytes = append(icoBytes, entryBytes...)
				offset = offset + imgLen
			}
			for _, img := range imgs {
				icoBytes = append(icoBytes, img...)
			}
			return uintptr(0)
		},
		nil,
	)

	// error out if non user enum error
	if err != nil && err != windows.ERROR_RESOURCE_ENUM_USER_STOP {
		// fmt.Println("failed enum")
		return nil, err
	}
	if innerErr != nil {
		return nil, innerErr
	}

	img, err := ico.Decode(bytes.NewBuffer(icoBytes))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// https://devblogs.microsoft.com/oldnewthing/20120720-00/?p=7083 algo at the end of the article
func getResource(hModule windows.Handle, lpType, lpName uintptr) (resPtr uintptr, resLen uint32, err error) {
	rInfo, err := winapi.FindResource(hModule, lpName, lpType)
	if err != nil {
		return 0, 0, err
	}
	if rInfo == 0 {
		return 0, 0, fmt.Errorf("no resource found")
	}

	resSize, err := winapi.SizeofResource(hModule, rInfo)
	if err != nil || resSize == 0 {
		return 0, 0, fmt.Errorf("zero size resource")
	}

	res, err := winapi.LoadResource(hModule, rInfo)
	if err != nil || res == 0 {
		return 0, 0, fmt.Errorf("couldn't load resource")
	}

	lockedRes, err := winapi.LockResource(res)
	if err != nil || res == 0 {
		return 0, 0, fmt.Errorf("couldn't lock resource")
	}
	return uintptr(lockedRes), resSize, nil
}

func encodeToBytes(i interface{}) ([]byte, error) {
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.LittleEndian, i)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
