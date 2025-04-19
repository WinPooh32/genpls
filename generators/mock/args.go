package mock

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

type config struct {
	Name string
	Pkg  string
	Dir  string
	Test bool
}

func (cfg *config) Filename() string {
	return filepath.Join(cfg.Dir, cfg.name())
}

func (cfg *config) name() string {
	const (
		suffixGo        = ".go"
		suffixGenGo     = "_gen.go"
		suffixGenTestGo = "_gen_test.go"
	)

	isNameGo := strings.HasSuffix(cfg.Name, suffixGo)
	isNameGenGo := strings.HasSuffix(cfg.Name, suffixGenGo)
	isNameGenTestGo := strings.HasSuffix(cfg.Name, suffixGenTestGo)

	if !isNameGenTestGo && !isNameGenGo && !isNameGo {
		return cfg.Name + suffixGenGo
	}

	return cfg.Name
}

func parseArgs(arguments []string, defaultValue config) (config, error) {
	var cfg config

	flagset := flag.NewFlagSet("", flag.ContinueOnError)

	flagset.StringVar(&cfg.Name, "name", defaultValue.Name, "file name")
	flagset.StringVar(&cfg.Pkg, "pkg", defaultValue.Pkg, "package name")
	flagset.StringVar(&cfg.Dir, "dir", defaultValue.Dir, "package dir path")
	flagset.BoolVar(&cfg.Test, "test", defaultValue.Test, "generate test package")

	if err := flagset.Parse(arguments); err != nil {
		return config{}, fmt.Errorf("flagset: Parse: %w", err)
	}

	return cfg, nil
}
