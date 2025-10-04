package util

import (
	"path/filepath"
	"strings"
)

// GetPathFromFile extracts the directory path from a full file path.
func GetPathFromFile(file string) string {
	return filepath.Dir(file)
}

// ReplaceExtOfFile replaces the file extension (including '.') with a new one.
// Example: ReplaceExtOfFile("a/b/test.txt", ".mp4") -> "a/b/test.mp4"
func ReplaceExtOfFile(file, ext string) string {
	oldExt := filepath.Ext(file)
	return strings.TrimSuffix(file, oldExt) + ext
}

// ReplacePathOfFile replaces the directory path of a file with a new one.
// Example: ReplacePathOfFile("a/b/test.txt", "x/y") -> "x/y/test.txt"
func ReplacePathOfFile(file, path string) string {
	return filepath.Join(path, filepath.Base(file))
}
