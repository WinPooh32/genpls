package genpls

import (
	"errors"
	"fmt"

	"golang.org/x/tools/go/packages"
)

type pkgerrs []*packages.Package

func (pkgs *pkgerrs) Error() string {
	var errs []error

	for _, pkg := range *pkgs {
		ee := make([]error, 0, len(pkg.Errors))
		for _, err := range pkg.Errors {
			ee = append(ee, err)
		}

		et := make([]error, 0, len(pkg.TypeErrors))
		for _, err := range pkg.TypeErrors {
			et = append(et, err)
		}

		perrs := []error{}
		if len(ee) > 0 {
			perrs = append(perrs, fmt.Errorf("\tmetadata: %w", errors.Join(ee...)))
		}

		if len(et) > 0 {
			perrs = append(perrs, fmt.Errorf("\ttypes: %w", errors.Join(et...)))
		}

		errs = append(errs, fmt.Errorf("package %s:\n%w", pkg.PkgPath, errors.Join(perrs...)))
	}

	return errors.Join(errs...).Error()
}
