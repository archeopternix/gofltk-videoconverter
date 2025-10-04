package util

import (
	"encoding/json"

	"fmt"
	"log/slog"
	"math"

	"path/filepath"
	"strconv"
	"strings"

	mediainfo "github.com/archeopternix/go-mediafileinfo"
)

const (
	binPath = "ffprobe.exe"
)

// IsVideo returns true is file is a video and optional checks container format
func IsVideo(fileURL string, container ...string) bool {
	avFormatCtx, err := mediainfo.GetMediaInfo(fileURL)
	if err != nil {
		slog.Debug("error in reading the AvFormatContext data", "file", fileURL, "msg", err)
		return false
	}

	// check the MIME type for video file
	mediaType := DetectMediaType(fileURL)
	if (mediaType != MediaTypeVideo) && (mediaType != MediaTypeUndefined) {
		slog.Debug("MIME type not video file", "file", fileURL)
		return false
	}

	for _, av := range avFormatCtx.Streams {
		if av.CodecParameters.CodecType == mediainfo.AVMEDIA_TYPE_VIDEO {
			return true
		}
	}

	slog.Debug("not a video file", "file", fileURL)
	return false
}

// containsAny checks if any value from the slice is contained in the given string
func containsAny(haystack string, needles []string) bool {
	strings.Split(haystack, ",")
	for _, needle := range needles {
		if strings.Contains(haystack, strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

type Info struct {
	Name        string
	FullPath    string
	FileType    string
	FileSize    string
	Duration    string
	Image       []byte
	VideoType   string
	ResolutionX int
	ResolutionY int
	FPS         string
	FieldOrder  string
}

func (i Info) String() string {
	jsonData, _ := json.Marshal(i)

	return string(jsonData)
}

// t.00g.00m.00k.000
func FormatNumberWithUnit(numberStr string) (string, error) {
	// Parse the input string to a float
	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number string: %v", err)
	}

	if number < 1000 {
		return fmt.Sprintf("%.0f", number), nil
	}

	// Define thresholds and units
	units := []string{"", "KB", "MB", "GB", "TB"}
	var unit string
	var value float64

	// Determine the appropriate unit
	for i := len(units) - 1; i >= 0; i-- {
		threshold := math.Pow(1000, float64(i))
		if number >= threshold {
			unit = units[i]
			value = number / threshold
			break
		}
	}

	// Format the value with a maximum of 2 decimal digits
	formattedValue := fmt.Sprintf("%.2f", value)

	// Return the formatted string
	return fmt.Sprintf("%s %s", formattedValue, unit), nil
}

func GetFirstVideoStream(avFormatCtx *mediainfo.AVFormatContext) (*mediainfo.AVStream, error) {
	for _, av := range avFormatCtx.Streams {
		if av.CodecParameters.CodecType == mediainfo.AVMEDIA_TYPE_VIDEO {
			return &av, nil
		}
	}
	slog.Debug("no video detected", "file", avFormatCtx.Filename)
	return nil, fmt.Errorf("no video in file: %s", avFormatCtx.Filename)
}

func GetInfoFromFileName(fileURL string) (*Info, error) {

	avFormatCtx, err := mediainfo.GetMediaInfo(fileURL)
	if err != nil {
		return nil, fmt.Errorf("error in reading the AvFormatContext data '%s': %v", fileURL, err)
	}

	path, _ := filepath.Abs(fileURL)

	info := &Info{FullPath: path, Name: filepath.Base(fileURL)}

	vstream, serr := GetFirstVideoStream(avFormatCtx)
	if serr != nil {
		return nil, fmt.Errorf("error reading video file '%s': %v", fileURL, serr)

	}

	info.FileType = vstream.CodecParameters.CodecIDText
	info.FileSize = avFormatCtx.FileSizeText
	info.VideoType = avFormatCtx.FormatLongName
	info.ResolutionX = vstream.CodecParameters.Width
	info.ResolutionY = vstream.CodecParameters.Height
	info.FPS = "n.a."
	info.Duration = vstream.DurationText
	info.FieldOrder = vstream.CodecParameters.FieldOrderText

	return info, nil

}

func CalculateDivision(input string, fieldorder string) (string, error) {
	// Split the input string by "/"
	parts := strings.Split(input, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid input format: %s", input)
	}

	// Parse the numerator and denominator
	numerator, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", fmt.Errorf("invalid numerator: %v", err)
	}

	denominator, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", fmt.Errorf("invalid denominator: %v", err)
	}

	// Check for division by zero
	if denominator == 0 {
		return "", fmt.Errorf("division by zero is not allowed")
	}

	// Perform the division
	result := float64(numerator) / float64(denominator)

	ret := fmt.Sprintf("%.2f", result)
	if strings.HasSuffix(ret, ".00") {
		ret, _ = strings.CutSuffix(ret, ".00")
	}

	if fieldorder == "progressive" {
		return fmt.Sprint(ret, "p"), nil
	}
	// Return the result as a string
	return fmt.Sprint(ret, "i"), nil
}

func ConvertSecondsToHMS(secondsStr string) (string, error) {
	// Parse the input string to a float
	seconds, err := strconv.ParseFloat(strings.TrimSpace(secondsStr), 64)
	if err != nil {
		return "", fmt.Errorf("invalid seconds string: %v", err)
	}

	// Convert seconds into hours, minutes, and seconds
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	remainingSeconds := int(seconds) % 60

	// Format the result
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, remainingSeconds), nil
}
