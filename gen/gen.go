package gen

import "context"

const CmdPrefix = "genpls:"

type Func func(ctx context.Context, name GeneratorName, pls []Please) ([]File, error)

type GeneratorName string

func (gn GeneratorName) Command() string {
	return CmdPrefix + string(gn)
}
