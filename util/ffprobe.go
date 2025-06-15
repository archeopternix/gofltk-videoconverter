package util

import (
	"encoding/json"

	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	. "gopkg.in/vansante/go-ffprobe.v2"
	"gopkg.in/yaml.v2"
)

const (
	binPath = "ffprobe.exe"
)

// FFprobe executes 'ffprobe.exe' and returns a populated ffprobe.ProbeData structure
func FFprobe(fileURL string, extraFFProbeOptions ...string) (*ProbeData, error) {
	args := append([]string{
		"-loglevel", "fatal",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
	}, extraFFProbeOptions...)

	// Add the file argument
	args = append(args, fileURL)

	data := exec.Command(binPath, args...)

	// Running the command and capturing the combined output (stdout and stderr)
	jsonData, err := data.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Conversion: %v Error %s", err, string(jsonData))
	}

	probe := &ProbeData{}

	// Unmarshal the struct into JSON
	err = json.Unmarshal(jsonData, probe)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %v\n", err)
	}

	return probe, nil
}

// IsVideo returns true is file is a video and optional checks container format
func IsVideo(fileURL string, container ...string) bool {
	probeData, err := FFprobe(fileURL)
	if err != nil {
		slog.Debug("error in reading the ffprobe data", "file", fileURL, "msg", err)
		return false
	}

	// check the MIME type for video file
	mediaType := DetectMediaType(fileURL)
	if (mediaType != MediaTypeVideo) && (mediaType != MediaTypeUndefined) {
		slog.Debug("MIME type not video file", "file", fileURL)
		return false
	}

	// checks if there is a first video stream
	if probeData.FirstVideoStream() != nil {
		// when container has a defined list of video formats and matches Format.Formatname
		if len(container) > 0 {
			if containsAny(probeData.Format.FormatName, container) {
				return true
			}
			slog.Debug("container format does not match", "file", fileURL, "video", probeData.Format.FormatName, "expected", container)
			return false
		}

		return true
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

// ProbeDataJson prints ffprobe.ProbeData as JSON
func ProbeDataJson(pd *ProbeData) []byte {
	// Marshal the struct into JSON
	jsonData, err := json.Marshal(pd.Format)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return nil
	}
	str := jsonData

	for _, s := range pd.Streams {
		jsonData, err := json.Marshal(s)
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			return nil
		}
		str = append(str, jsonData...)
	}
	for _, c := range pd.Chapters {
		jsonData, err := json.Marshal(c)
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			return nil
		}
		str = append(str, jsonData...)
	}
	return str
}

// ProbeDataJson prints ffprobe.ProbeData as YAML
func ProbeDataYAML(pd *ProbeData) []byte {
	// Marshal the struct into JSON
	yamlData, err := yaml.Marshal(pd.Format)
	if err != nil {
		fmt.Printf("Error marshaling Format YAML: %v\n", err)
		return nil
	}
	str := []byte("format:\n")
	str = append(str, yamlData...)

	str = append(str, []byte("streams:\n")...)
	for _, s := range pd.Streams {
		yamlData, err := yaml.Marshal(s)
		if err != nil {
			fmt.Printf("Error marshaling Streams YAML: %v\n", err)
			return nil
		}

		str = append(str, yamlData...)
	}
	str = append(str, []byte("chapters:\n")...)
	for _, c := range pd.Chapters {
		yamlData, err := yaml.Marshal(c)
		if err != nil {
			fmt.Printf("Error marshaling Chapters YAML: %v\n", err)
			return nil
		}
		str = append(str, yamlData...)
	}
	return str
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

func GetInfoFromFileName(fileURL string) *Info {
	probeData, err := FFprobe(fileURL)
	if err != nil {
		return nil
	}

	path, _ := filepath.Abs(fileURL)

	info := &Info{FullPath: path, Name: filepath.Base(fileURL)}

	// checks if there is a first video stream
	if probeData.FirstVideoStream() != nil {
		info.FileType = probeData.Format.FormatName
		info.FileSize = probeData.Format.Size
		info.FileSize, _ = FormatNumberWithUnit(info.FileSize)
		info.VideoType = probeData.FirstVideoStream().CodecName
		info.ResolutionX = probeData.FirstVideoStream().Width
		info.ResolutionY = probeData.FirstVideoStream().Height
		info.FPS, _ = CalculateDivision(probeData.FirstVideoStream().AvgFrameRate, probeData.FirstVideoStream().FieldOrder)
		info.Duration, _ = ConvertSecondsToHMS(probeData.FirstVideoStream().Duration)
		info.FieldOrder = probeData.FirstVideoStream().FieldOrder
	}

	return info

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
