package genpls_test

import (
	"testing"

	"github.com/WinPooh32/genpls"
	"github.com/stretchr/testify/assert"
)

func TestPlease_FmtFileName(t *testing.T) {
	t.Parallel()

	type args struct {
		name genpls.GeneratorName
	}

	tests := []struct {
		name string
		pls  genpls.Please
		args args
		want string
	}{
		{
			name: "usual file",
			pls: genpls.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{genpls.GeneratorName("stub")},
			want: "/pkgname/file_stub_gen.go",
		},
		{
			name: "test file",
			pls: genpls.Please{
				Filename: "/pkgname/file_test.go",
			},
			args: args{genpls.GeneratorName("stub")},
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
		name genpls.GeneratorName
		test bool
	}

	tests := []struct {
		name         string
		pls          genpls.Please
		args         args
		wantFilename string
	}{
		{
			name: "command name generated file",
			pls: genpls.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{
				name: genpls.GeneratorName("mock"),
				test: false,
			},
			wantFilename: "/pkgname/mock_gen.go",
		},
		{
			name: "command name generated test file",
			pls: genpls.Please{
				Filename: "/pkgname/file.go",
			},
			args: args{
				name: genpls.GeneratorName("mock"),
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
