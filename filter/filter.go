package filter

type Filter struct {
	Name      string
	InFiles   []string          // full path included
	OutFiles  []string          // just the file name
	OutDir    string            // directory for output files
	Params    map[string]string // additional global parameters for filter
	Optionals []string          // optional per file parameter
}

func NewFilter(name string) *Filter {
	f := &Filter{
		Name:   name,
		Params: make(map[string]string),
	}
	return f
}
