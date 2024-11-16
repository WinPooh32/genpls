package xslices_test

import (
	"slices"
	"testing"

	. "github.com/WinPooh32/genpls/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func Test_split(t *testing.T) {
	t.Parallel()

	type args struct {
		s []string
		n int
	}

	tests := []struct {
		name string
		args args
		want [][]string
	}{
		// 1 is the special case
		{"len(s) = 1 n = 1", args{[]string{"a"}, 1}, [][]string{{"a"}}},
		{"len(s) = 2 n = 1", args{[]string{"a", "b"}, 1}, [][]string{{"a", "b"}}},
		{"len(s) = 3 n = 1", args{[]string{"a", "b", "c"}, 1}, [][]string{{"a", "b", "c"}}},
		// Even n
		{"len(s) = 1 n = 2", args{[]string{"a"}, 2}, [][]string{{"a"}}},
		{"len(s) = 2 n = 2", args{[]string{"a", "b"}, 2}, [][]string{{"a"}, {"b"}}},
		{"len(s) = 3 n = 2", args{[]string{"a", "b", "c"}, 2}, [][]string{{"a", "b"}, {"c"}}},
		{"len(s) = 4 n = 2", args{[]string{"a", "b", "c", "d"}, 2}, [][]string{{"a", "b"}, {"c", "d"}}},
		{"len(s) = 5 n = 2", args{[]string{"a", "b", "c", "d", "e"}, 2}, [][]string{{"a", "b", "c"}, {"d", "e"}}},
		// Odd n
		{"len(s) = 1 n = 3", args{[]string{"a"}, 3}, [][]string{{"a"}}},
		{"len(s) = 2 n = 3", args{[]string{"a", "b"}, 3}, [][]string{{"a"}, {"b"}}},
		{"len(s) = 3 n = 3", args{[]string{"a", "b", "c"}, 3}, [][]string{{"a"}, {"b"}, {"c"}}},
		{"len(s) = 4 n = 3", args{[]string{"a", "b", "c", "d"}, 3}, [][]string{{"a"}, {"b"}, {"c", "d"}}},
		{"len(s) = 5 n = 3", args{[]string{"a", "b", "c", "d", "e"}, 3}, [][]string{{"a", "b"}, {"c", "d"}, {"e"}}},

		{"len(s) = 5 n = 5", args{[]string{"a", "b", "c", "d", "e"}, 5}, [][]string{{"a"}, {"b"}, {"c"}, {"d"}, {"e"}}},
		{
			"len(s) = 6 n = 5",
			args{[]string{"a", "b", "c", "d", "e", "f"}, 5},
			[][]string{{"a"}, {"b"}, {"c"}, {"d"}, {"e", "f"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotParts := slices.Collect(Split(tt.args.s, tt.args.n))

			assert.Equal(t, tt.want, gotParts)
		})
	}
}
