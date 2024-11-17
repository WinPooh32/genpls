package genpls

import (
	"path/filepath"
	"strings"
)

const testSuffix = "_test"

type Please struct {
	Filename string
	Args     []string
	TS       *TypeSpec
}

// FmtFileName formats absolute path for a new destination file.
func (pls *Please) FmtFileName(name GeneratorName) (filename string) {
	dir := filepath.Dir(pls.Filename)

	basename := strings.TrimSuffix(filepath.Base(pls.Filename), ".go")

	var suffix string

	if strings.HasSuffix(basename, testSuffix) {
		basename = strings.TrimSuffix(basename, testSuffix)
		suffix = "_gen_test.go"
	} else {
		suffix = "_gen.go"
	}

	filename = filepath.Join(dir, basename+"_"+string(name)+suffix)

	return filename
}

// FmtFileName formats absolute path for a new destination file.
func (pls *Please) FmtGeneratorFileName(name GeneratorName, test bool) (filename string) {
	dir := filepath.Dir(pls.Filename)

	if test {
		filename = filepath.Join(dir, string(name)+"_gen_test.go")
	} else {
		filename = filepath.Join(dir, string(name)+"_gen.go")
	}

	return filename
}
