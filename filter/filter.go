package filter

type Filter struct {
	Name        string
	InFile      []string          // full path included
	OutFilename []string          // just the file name
	OutDir      string            // directory for output files
	Params      map[string]string // additional global parameters for filter
	Optional    []string          // optional per file parameter
}

func NewFilter(name string) *Filter {
	f := &Filter{
		Name:   name,
		Params: make(map[string]string),
	}
	return f
}
