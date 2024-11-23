package gen_test

import (
	"testing"

	"github.com/WinPooh32/genpls/gen"
	"github.com/stretchr/testify/assert"
)

func TestPlease_FmtFileName(t *testing.T) {
	t.Parallel()

	type args struct {
		name gen.GeneratorName
	}

	tests := []struct {
		name string
		pls  gen.Please
		args args
		want string
	}{
		{
			name: "usual file",
			pls: gen.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{gen.GeneratorName("stub")},
			want: "/pkgname/file_stub_gen.go",
		},
		{
			name: "test file",
			pls: gen.Please{
				Filename: "/pkgname/file_test.go",
			},
			args: args{gen.GeneratorName("stub")},
			want: "/pkgname/file_stub_gen_test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pls := tt.pls
			assert.Equal(t, tt.want, pls.FormatFileName(tt.args.name))
		})
	}
}

func TestPlease_FmtGeneratorFileName(t *testing.T) {
	t.Parallel()

	type args struct {
		name gen.GeneratorName
		test bool
	}

	tests := []struct {
		name         string
		pls          gen.Please
		args         args
		wantFilename string
	}{
		{
			name: "command name generated file",
			pls: gen.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{
				name: gen.GeneratorName("mock"),
				test: false,
			},
			wantFilename: "/pkgname/mock_gen.go",
		},
		{
			name: "command name generated test file",
			pls: gen.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{
				name: gen.GeneratorName("mock"),
				test: true,
			},
			wantFilename: "/pkgname/mock_gen_test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pls := tt.pls
			assert.Equal(t, tt.wantFilename, pls.FormatGeneratorFileName(tt.args.name, tt.args.test))
		})
	}
}
