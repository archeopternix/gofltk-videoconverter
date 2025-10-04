package util

import "testing"

func TestGetPathFromFile(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"a/b/c.txt", "a/b"},
		{"/usr/local/bin/test", "/usr/local/bin"},
		{"file.txt", "."},
	}

	for _, tt := range tests {
		got := GetPathFromFile(tt.in)
		if got != tt.want {
			t.Errorf("GetPathFromFile(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestReplaceExtOfFile(t *testing.T) {
	tests := []struct {
		file string
		ext  string
		want string
	}{
		{"a/b/c.txt", ".mp4", "a/b/c.mp4"},
		{"a/b/c", "_test.pdf", "a/b/c_test.pdf"},
		{"a/b/c.tar.gz", ".zip", "a/b/c.tar.zip"},
	}

	for _, tt := range tests {
		got := ReplaceExtOfFile(tt.file, tt.ext)
		if got != tt.want {
			t.Errorf("ReplaceExtOfFile(%q, %q) = %q; want %q", tt.file, tt.ext, got, tt.want)
		}
	}
}

func TestReplacePathOfFile(t *testing.T) {
	tests := []struct {
		file string
		path string
		want string
	}{
		{"a/b/c.txt", "x/y", "x/y/c.txt"},
		{"/usr/local/bin/test", "/tmp", "/tmp/test"},
		{"file.txt", "newdir", "newdir/file.txt"},
	}

	for _, tt := range tests {
		got := ReplacePathOfFile(tt.file, tt.path)
		if got != tt.want {
			t.Errorf("ReplacePathOfFile(%q, %q) = %q; want %q", tt.file, tt.path, got, tt.want)
		}
	}
}
