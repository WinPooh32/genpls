package genpls

import (
	"iter"
	"maps"
)

// File is a result of the code generation.
type File struct {
	// Name is an absolute file path.
	Name string
	// Data is a file content.
	Data []byte
}

// IterateFiles returns iterator of grouped commands by the filename.
func IterateFiles(pls []Please) iter.Seq2[string, []Please] {
	m := map[string][]Please{}

	for _, p := range pls {
		m[p.Filename] = append(m[p.Filename], p)
	}

	return maps.All(m)
}
