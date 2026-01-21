//go:build linux
// +build linux

package geticon

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
)

// FromPid returns the app icon of the app currently running
// on the given pid.
func FromPid(pid uint32) (image.Image, error) {
	exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return nil, fmt.Errorf("failed to gather icon")
	}
	return FromPath(exePath)
}

// FromPath returns the app icon app at the specified path.
func FromPath(appPath string) (image.Image, error) {
	absPath, _ := filepath.Abs(appPath)

	// try to find icon from .desktop file
	if iconName := findIconName(absPath); iconName != "" {
		if img, err := loadIcon(iconName); err == nil {
			return img, nil
		}
	}

	// try executable basename as icon name
	baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	if img, err := loadIcon(baseName); err == nil {
		return img, nil
	}
	if img, err := loadIcon(strings.ToLower(baseName)); err == nil {
		return img, nil
	}

	return nil, fmt.Errorf("failed to gather icon")
}

func findIconName(exePath string) string {
	dirs := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/applications"),
		"/usr/share/applications",
		"/usr/local/share/applications",
	}

	exeBase := filepath.Base(exePath)
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.desktop"))
		for _, file := range files {
			if icon := parseDesktopFile(file, exePath, exeBase); icon != "" {
				return icon
			}
		}
	}
	return ""
}

func parseDesktopFile(path, exePath, exeBase string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var icon string
	var match bool
	inEntry := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		switch key {
		case "Icon":
			icon = val
		case "Exec":
			cmd := strings.Fields(val)
			if len(cmd) > 0 {
				// skip env prefix (e.g., "env VAR=val /usr/bin/app")
				i := 0
				if cmd[0] == "env" {
					for i = 1; i < len(cmd); i++ {
						if !strings.Contains(cmd[i], "=") {
							break
						}
					}
				}
				if i < len(cmd) && (cmd[i] == exePath || filepath.Base(cmd[i]) == exeBase) {
					match = true
				}
			}
		case "TryExec":
			if val == exePath || filepath.Base(val) == exeBase {
				match = true
			}
		}
	}

	if match && icon != "" {
		return icon
	}
	return ""
}

func loadIcon(name string) (image.Image, error) {
	if name == "" {
		return nil, fmt.Errorf("empty icon name")
	}

	// absolute path
	if filepath.IsAbs(name) {
		return loadIconFile(name)
	}

	// search icon theme directories
	sizes := []string{"512x512", "256x256", "128x128", "96x96", "64x64", "48x48"}
	contexts := []string{"apps", "applications"}
	baseDirs := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/icons"),
		filepath.Join(os.Getenv("HOME"), ".icons"),
		"/usr/share/icons",
		"/usr/share/pixmaps",
	}

	// try hicolor theme first
	for _, baseDir := range baseDirs {
		for _, size := range sizes {
			for _, ctx := range contexts {
				for _, ext := range []string{".png", ".ico", ".bmp"} {
					p := filepath.Join(baseDir, "hicolor", size, ctx, name+ext)
					if img, err := loadIconFile(p); err == nil {
						return img, nil
					}
				}
			}
		}
	}

	// try pixmaps fallback
	for _, ext := range []string{".png", ".ico", ".bmp", ".xpm"} {
		for _, baseDir := range baseDirs {
			if img, err := loadIconFile(filepath.Join(baseDir, name+ext)); err == nil {
				return img, nil
			}
		}
	}

	return nil, fmt.Errorf("icon not found")
}

func loadIconFile(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if isPNG(data) {
		return png.Decode(bytes.NewReader(data))
	}
	if isICO(data) {
		return decodeICO(data)
	}
	return bmp.Decode(bytes.NewReader(data))
}

func isICO(b []byte) bool {
	return len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0
}

func decodeICO(data []byte) (image.Image, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("invalid ico")
	}
	count := binary.LittleEndian.Uint16(data[4:6])
	if count == 0 {
		return nil, fmt.Errorf("invalid ico")
	}

	// find largest image (6 byte header + 16 byte entries)
	var bestOff, bestLen uint32
	var bestSize int
	for i := 0; i < int(count); i++ {
		off := 6 + i*16
		if off+16 > len(data) {
			break
		}
		e := data[off:]
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w*h > bestSize {
			bestSize = w * h
			bestLen = binary.LittleEndian.Uint32(e[8:12])
			bestOff = binary.LittleEndian.Uint32(e[12:16])
		}
	}

	if bestLen == 0 || int(bestOff+bestLen) > len(data) {
		return nil, fmt.Errorf("invalid ico")
	}
	imgData := data[bestOff : bestOff+bestLen]

	if isPNG(imgData) {
		return png.Decode(bytes.NewReader(imgData))
	}
	if len(imgData) < 40 {
		return nil, fmt.Errorf("invalid ico")
	}
	// BMP without file header
	hdr := []byte{'B', 'M', 0, 0, 0, 0, 0, 0, 0, 0, 54, 0, 0, 0}
	return bmp.Decode(bytes.NewReader(append(hdr, imgData...)))
}

