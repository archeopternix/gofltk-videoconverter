package util

import (
	"net/http"
	"os"
	"strings"
)

type MediaType int

const (
	MediaTypeUnknown MediaType = iota
	MediaTypeAudio
	MediaTypeVideo
	MediaTypeUndefined
)

func (m MediaType) String() string {
	switch m {
	case MediaTypeAudio:
		return "Audio"
	case MediaTypeVideo:
		return "Video"
	default:
		return "Unknown"
	}
}

func DetectMediaType(path string) MediaType {
	file, err := os.Open(path)
	if err != nil {
		return MediaTypeUnknown
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return MediaTypeUnknown
	}

	mime := http.DetectContentType(buffer[:n])
	switch {
	case strings.HasPrefix(mime, "audio/"):
		return MediaTypeAudio
	case strings.HasPrefix(mime, "video/"):
		return MediaTypeVideo
	case strings.HasSuffix(mime, "/octet-stream"):
		return MediaTypeUndefined
	default:
		return MediaTypeUnknown
	}
}
