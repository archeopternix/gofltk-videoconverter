package media

type ScanType string

const (
	ScanProgressive ScanType = "progressive"
	ScanInterlaced  ScanType = "interlaced"
	ScanUnknown     ScanType = "unknown"
)

type FieldOrder string

const (
	FieldOrderTFF     FieldOrder = "tff"
	FieldOrderBFF     FieldOrder = "bff"
	FieldOrderUnknown FieldOrder = "unknown"
)

type MediaSpec struct {
	Container   string
	Codec       string
	Width       int
	Height      int
	FPS         float64
	ScanType    ScanType
	FieldOrder  FieldOrder
	PixelFormat string
	ColorSpace  string
	ColorFamily string
	IsYV12      bool
	AudioCodec  string
}

// WithScanFallback assumes progressive video unless both interlacing and its
// field order are known. Progressive video does not need a field order.
func (s MediaSpec) WithScanFallback() MediaSpec {
	if s.ScanType != ScanInterlaced || (s.FieldOrder != FieldOrderTFF && s.FieldOrder != FieldOrderBFF) {
		s.ScanType = ScanProgressive
		s.FieldOrder = FieldOrderUnknown
	}
	return s
}

type ArtifactType string

const (
	ArtifactSource   ArtifactType = "source"
	ArtifactAviSynth ArtifactType = "avisynth"
	ArtifactVideo    ArtifactType = "video"
)

type Artifact struct {
	Path  string
	Type  ArtifactType
	Media MediaSpec
}
