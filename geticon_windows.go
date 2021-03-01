package geticon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"unsafe"

	"github.com/fcjr/geticon/internal/winapi"
	"golang.org/x/image/bmp"
	"golang.org/x/sys/windows"
)

const pngHeader = "\x89PNG\r\n\x1a\n"

// https://devblogs.microsoft.com/oldnewthing/20120720-00/?p=7083

type grpIconDirHeader struct {
	IDReserved uint16
	IDType     uint16
	IDCount    uint16
}
type grpIconResourceDirEntry struct {
	BWidth       byte
	BHeight      byte
	BColorCount  byte
	BReserved    byte
	WPlanes      uint16
	WBitCount    uint16
	DWBytesInRes uint32
	NID          uint16
}
type grpIconDiskDirEntry struct {
	BWidth        byte
	BHeight       byte
	BColorCount   byte
	BReserved     byte
	WPlanes       uint16
	WBitCount     uint16
	DWBytesinRes  uint32
	DWImageOffset uint32
}

// http://www.ece.ualberta.ca/~elliott/ee552/studentAppNotes/2003_w/misc/bmp_file_format/bmp_file_format.htm
type bitmapFileHeader struct {
	Signature       [2]byte
	FileSize        uint32
	Reserved        uint32
	DataOffset      uint32
	Size            uint32
	Width           uint32
	Height          uint32
	Planes          uint16
	BitsPerPixel    uint16
	Compression     uint32
	ImageSize       uint32
	XpixelsPerM     uint32
	YpixelsPerM     uint32
	ColorsUsed      uint32
	ImportantColors uint32
}

// FromPid returns the app icon of the app currently running
// on the given pid.
func FromPid(pid uint32) (image.Image, error) {
	// get path from pid
	exePath, err := winapi.QueryFullProcessImageNameW(pid, 0)
	if err != nil {
		return nil, err
	}

	return FromPath(exePath)
}

// FromPath returns the app icon app at the specified path.
func FromPath(exePath string) (image.Image, error) {

	// get handle
	exeHandle, err := winapi.LoadLibraryEx(
		exePath,
		winapi.LoadLibraryAsImageResource|winapi.LoadLibraryAsDatafile,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(exeHandle)

	var img image.Image
	var innerErr error
	// enum rt_group_icons and grab the first one
	err = winapi.EnumResourceNamesA(
		exeHandle,
		winapi.MakeIntResource(winapi.RtGroupIcon),
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

			iconHeader := *(*grpIconDirHeader)(unsafe.Pointer(resPtr))

			// find best icon
			entriesPtr := resPtr + 6
			bestEntry := *(*grpIconResourceDirEntry)(unsafe.Pointer(entriesPtr))
			for i := 0; i < int(iconHeader.IDCount); i++ {
				entry := *(*grpIconResourceDirEntry)(unsafe.Pointer(entriesPtr + uintptr(i*14)))

				eW := entry.BWidth
				eH := entry.BHeight
				bW := bestEntry.BWidth
				bH := bestEntry.BHeight
				// find best icon
				if (eW == 0 || (eW > bW && bW != 0)) && // 0 is actually 256 b/c its stored in a byte
					(eH == 0 || (eH > bH && bH != 0)) {
					bestEntry = entry
				}
			}

			// get best image
			bestPtr, bestLen, err := getResource(hModule, winapi.RtIcon, uintptr(bestEntry.NID))
			if err != nil {
				innerErr = err
				return uintptr(0)
			}
			if bestLen == 0 {
				innerErr = fmt.Errorf("best image had was zero bytes")
				return uintptr(0)
			}

			tmpData := cToGoSlice(unsafe.Pointer(bestPtr), int(bestLen))
			// TODO figure out a better way to copy this
			imgData := make([]byte, bestLen)
			for i, b := range tmpData {
				imgData[i] = b
			}

			if isPNG(imgData) {
				img, innerErr = png.Decode(bytes.NewBuffer(imgData))
				return uintptr(0)
			}
			// if not png then it must be a bitmap

			// ico just contains the raw bitmap data so
			// to encode it with the standard library we must
			// create a file header

			if len(imgData) < 40 {
				innerErr = fmt.Errorf("invalid bitmap data")
				return uintptr(0)
			}
			// cut the InfoHeader from imgData and build
			// our own b/c for some reason its not working
			// TODO figure this out
			imgData = imgData[40:]

			var dataOffset uint32
			if bestEntry.BColorCount == 0 && bestEntry.WBitCount <= 8 {
				dataOffset = 14 + 40 + 4*(1<<bestEntry.WBitCount)
			} else {
				dataOffset = 14 + 40 + 4*uint32(bestEntry.BColorCount)
			}

			bmpHeader := &bitmapFileHeader{
				Signature:    [2]byte{'B', 'M'},
				FileSize:     14 + 40 + uint32(len(imgData)),
				DataOffset:   dataOffset,
				Size:         40,
				Width:        uint32(bestEntry.BWidth),
				Height:       uint32(bestEntry.BHeight),
				Planes:       bestEntry.WPlanes,
				BitsPerPixel: bestEntry.WBitCount,
				ColorsUsed:   uint32(bestEntry.BColorCount),
			}

			headerBytes, err := encodeToBytes(bmpHeader)
			if err != nil {
				innerErr = err
				return uintptr(0)
			}
			imgData = append(headerBytes, imgData...)

			// TODO the builtin bmp package doesn't support
			// 1 = monochrome palette or
			// 4 = 4bit palletized
			img, innerErr = bmp.Decode(bytes.NewBuffer(imgData))
			if innerErr != nil {
				fmt.Println(innerErr)
			}
			return uintptr(0)
		},
		nil,
	)
	// error out if non user enum error
	if err != nil && err != windows.ERROR_RESOURCE_ENUM_USER_STOP {
		return nil, err
	}
	return img, innerErr
}

// https://devblogs.microsoft.com/oldnewthing/20120720-00/?p=7083 algo at the end of the article
func getResource(hModule windows.Handle, lpType, lpName uintptr) (resPtr uintptr, resLen uint32, err error) {
	rInfo, err := winapi.FindResourceA(hModule, lpName, lpType)
	if err != nil {
		return 0, 0, err
	}
	if rInfo == 0 {
		return 0, 0, fmt.Errorf("no resource found")
	}

	resSize, err := winapi.SizeofResource(hModule, rInfo)
	if err != nil {
		return 0, 0, err
	}
	if resSize == 0 {
		return 0, 0, fmt.Errorf("zero size resource")
	}

	res, err := winapi.LoadResource(hModule, rInfo)
	if err != nil {
		return 0, 0, err
	}
	if res == 0 {
		return 0, 0, fmt.Errorf("couldn't load resource")
	}

	lockedRes, err := winapi.LockResource(res)
	if err != nil {
		return 0, 0, err
	}
	if res == 0 {
		return 0, 0, fmt.Errorf("couldn't lock resource")
	}

	return uintptr(lockedRes), resSize, nil
}

func isPNG(b []byte) bool {
	return len(b) >= 8 && string(b[0:8]) == pngHeader
}

// extracts complete ico from from exe at exePath this was
// previously used for this library but is now just here for reference
func getCompleteIcoFromPath(exePath string) ([]byte, error) {
	// get handle
	exeHandle, err := winapi.LoadLibraryEx(
		exePath,
		winapi.LoadLibraryAsImageResource|winapi.LoadLibraryAsDatafile,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(exeHandle)

	// var img image.Image
	icoBytes := []byte{}

	var innerErr error
	// enum rt_group_icons and grab the first one
	err = winapi.EnumResourceNamesA(
		exeHandle,
		winapi.MakeIntResource(winapi.RtGroupIcon),
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

			iconHeader := *(*grpIconDirHeader)(unsafe.Pointer(resPtr))

			// get entry info
			entriesPtr := resPtr + 6
			var entries []grpIconResourceDirEntry
			for i := 0; i < int(iconHeader.IDCount); i++ {
				entry := *(*grpIconResourceDirEntry)(unsafe.Pointer(entriesPtr + uintptr(i*14)))
				entries = append(entries, entry)
			}

			// get icons
			imgs := [][]byte{}
			for _, entry := range entries {
				// fmt.Printf("%+v\n", entry)
				imgPntr, imgLen, err := getResource(
					hModule,
					winapi.RtIcon,
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
				dirEntry := grpIconDiskDirEntry{
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

	return icoBytes, nil
}

func encodeToBytes(i interface{}) ([]byte, error) {
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.LittleEndian, i)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
