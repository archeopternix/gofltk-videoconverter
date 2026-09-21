package workflow

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
	"gopkg.in/yaml.v3"
)

type Catalog struct {
	workflows []*Definition
}

func LoadCatalog(directory string) (*Catalog, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read workflow directory %q: %w", directory, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yaml" || ext == ".yml" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	catalog := &Catalog{}
	for _, name := range names {
		filename := filepath.Join(directory, name)
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read workflow %q: %w", filename, err)
		}
		var definition Definition
		if err := yaml.Unmarshal(data, &definition); err != nil {
			return nil, fmt.Errorf("parse workflow %q: %w", filename, err)
		}
		if definition.Version == 0 || definition.ID == "" || definition.Name == "" || len(definition.Workflow) == 0 {
			return nil, fmt.Errorf("workflow %q requires version, id, name and workflow", filename)
		}
		definition.Source = filename
		catalog.workflows = append(catalog.workflows, &definition)
	}
	return catalog, nil
}

func (c *Catalog) Match(spec media.MediaSpec) (*Definition, error) {
	type candidate struct {
		definition  *Definition
		specificity int
	}
	var candidates []candidate
	for _, definition := range c.workflows {
		if matches(definition.Match, spec) {
			candidates = append(candidates, candidate{
				definition:  definition,
				specificity: specificity(definition.Match),
			})
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no workflow matches %dx%d %.3f fps %s %s", spec.Width, spec.Height, spec.FPS, spec.ScanType, spec.Codec)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].specificity != candidates[j].specificity {
			return candidates[i].specificity > candidates[j].specificity
		}
		return candidates[i].definition.Priority > candidates[j].definition.Priority
	})
	best := candidates[0]
	if len(candidates) > 1 && candidates[1].specificity == best.specificity && candidates[1].definition.Priority == best.definition.Priority {
		return nil, fmt.Errorf("ambiguous workflows %q and %q", best.definition.ID, candidates[1].definition.ID)
	}
	return best.definition, nil
}

func matches(match MatchDefinition, spec media.MediaSpec) bool {
	if match.Interlaced != nil {
		if spec.ScanType == media.ScanUnknown || (*match.Interlaced != (spec.ScanType == media.ScanInterlaced)) {
			return false
		}
	}
	if match.Width != 0 && match.Width != spec.Width {
		return false
	}
	if match.Height != 0 && match.Height != spec.Height {
		return false
	}
	if len(match.FPS) > 0 {
		tolerance := match.FPSTolerance
		if tolerance == 0 {
			tolerance = 0.01
		}
		matched := false
		for _, fps := range match.FPS {
			if math.Abs(fps-spec.FPS) <= tolerance {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if !matchesOne(match.PixelFormat, spec.PixelFormat) {
		return false
	}
	if len(match.ColorSpace) > 0 && !matchesOne(match.ColorSpace, spec.ColorSpace) && !matchesOne(match.ColorSpace, spec.ColorFamily) {
		return false
	}
	return matchesOne(match.Codec, spec.Codec)
}

func matchesOne(allowed []string, actual string) bool {
	if len(allowed) == 0 {
		return true
	}
	actual = strings.ToLower(strings.TrimSpace(actual))
	for _, value := range allowed {
		if strings.ToLower(strings.TrimSpace(value)) == actual {
			return true
		}
	}
	return false
}

func specificity(match MatchDefinition) int {
	result := 0
	if match.Interlaced != nil {
		result++
	}
	if match.Width != 0 || match.Height != 0 {
		result++
	}
	if len(match.FPS) > 0 {
		result++
	}
	if len(match.PixelFormat) > 0 || len(match.ColorSpace) > 0 {
		result++
	}
	if len(match.Codec) > 0 {
		result++
	}
	return result
}
