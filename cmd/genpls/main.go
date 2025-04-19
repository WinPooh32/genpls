package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/WinPooh32/genpls"
	"github.com/WinPooh32/genpls/gen"
	"github.com/WinPooh32/genpls/generators/mock"
	"github.com/WinPooh32/genpls/generators/proxy"
	"github.com/WinPooh32/genpls/generators/stub"
)

// Enabled generators.
var generators = map[gen.GeneratorName]gen.Func{
	"stub":  stub.Generate,
	"proxy": proxy.Generate,
	"mock":  mock.Generate,
}

type argSet []string

func (a *argSet) String() string {
	return strings.Join(*a, ", ")
}

func (a *argSet) Set(s string) error {
	*a = strings.Split(s, ",")
	return nil
}

type flags struct {
	jobs     int
	dir      string
	patterns argSet
}

func main() {
	var flags flags

	flag.IntVar(&flags.jobs, "jobs", 0, "parallel jobs number")
	flag.StringVar(&flags.dir, "dir", "", "go module dir")
	flag.Var(&flags.patterns, "pattern", "list of package patterns")
	flag.Parse()

	if len(flags.patterns) == 0 {
		flags.patterns = []string{"./..."}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := generate(ctx, flags); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func generate(ctx context.Context, flags flags) error {
	gen, err := genpls.NewGenerator()
	if err != nil {
		return fmt.Errorf("new generator: %w", err)
	}

	_, err = gen.Load(ctx, flags.dir, flags.patterns...)
	if err != nil {
		return fmt.Errorf("load source files to the generator: %w", err)
	}

	filesCh := gen.Generate(ctx, flags.jobs, generators)

	for file := range filesCh {
		if err := file.Err; err != nil {
			return fmt.Errorf("generate: %w", err)
		}

		if err := writeFile(file.Ok.Name, file.Ok.Data); err != nil {
			return fmt.Errorf("write file at %q: %w", file.Ok.Name, err)
		}
	}

	return nil
}

func writeFile(name string, data []byte) (err error) {
	if err := os.MkdirAll(filepath.Dir(name), os.ModePerm); err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	defer func() {
		err = errors.Join(err, file.Close())
	}()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
