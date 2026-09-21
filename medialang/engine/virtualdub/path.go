package virtualdub

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
)

type PathConverter struct {
	Mode      string
	WineDrive string
	Mappings  []config.WindowsPathMap
}

func NewPathConverter(tool config.VirtualDubTool) *PathConverter {
	mappings := append([]config.WindowsPathMap(nil), tool.Mappings...)
	sort.SliceStable(mappings, func(i, j int) bool {
		return len(mappings[i].Source) > len(mappings[j].Source)
	})
	return &PathConverter{Mode: tool.PathMode, WineDrive: tool.WineDrive, Mappings: mappings}
}

func (c *PathConverter) ToWindowsPath(path string) (string, error) {
	if isWindowsPath(path) {
		return strings.ReplaceAll(path, "/", `\`), nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for _, mapping := range c.Mappings {
		relative, err := filepath.Rel(mapping.Source, abs)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		target := strings.TrimRight(mapping.Target, `\/`)
		if relative == "." {
			return target, nil
		}
		return target + `\` + strings.ReplaceAll(relative, "/", `\`), nil
	}

	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode == "" {
		mode = "windows"
	}
	if mode == "windows" {
		if runtime.GOOS != "windows" {
			return "", fmt.Errorf("cannot convert POSIX path %q in windows mode", path)
		}
		return strings.ReplaceAll(abs, "/", `\`), nil
	}
	if mode != "wine" {
		return "", fmt.Errorf("unknown VirtualDub path mode %q", c.Mode)
	}

	drive := strings.TrimSpace(c.WineDrive)
	if drive == "" {
		drive = "Z:"
	}
	drive = strings.TrimRight(drive, `\/`)
	return drive + `\` + strings.ReplaceAll(strings.TrimLeft(filepath.ToSlash(abs), "/"), "/", `\`), nil
}

func isWindowsPath(path string) bool {
	return len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}
