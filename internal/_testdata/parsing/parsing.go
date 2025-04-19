package parse

import (
	"go/types"
	io_1 "io"

	types_1 "parse/types"
	types_2 "parse/types"
)

//genpls:test S1
type S1 struct {
	// S1Field1 doc
	S1Field1 string
	// S1Field2 doc
	S1Field2 int
}

//genpls:test s2 arg2 arg3
type (
	s2 struct {
		// S2Field1 doc
		s2Field1 string
		// s2Field2 doc
		s2Field2 int
	}
)

// S3 struct doc
//
//genpls:test S3
type S3 struct {
	// S3Field1
	// multiline
	// doc
	S3Field1 string
	S3Field2 int
}

// method1 doc
func (s S3) method1(a string, b string) (int, error) { return 0, nil }

// Method2 doc
func (s *S3) Method2() {}

// method3 doc
func (S3) method3() error { return nil }

// method4 doc
func (*S3) method4() {}

// method5 doc
func (_ S3) method5() {}

// method6 doc
func (_ *S3) method6() {}

//genpls:test S4
type S4[T any] struct {
	// S4Field doc
	S4Field T
}

// method7 doc
func (S4[T]) method7(a T) {}

//genpls:test I1
//genpls:proxy
//genpls:mock
type I1 interface {
	// IMethod1 doc
	IMethod1()
	// imethod2 doc
	imethod2()
}

//genpls:stub
//genpls:proxy
//genpls:mock
type I2[T any, U comparable, Q io_1.Reader] interface {
	IMethod1()
	imethod2(t T) (u U)
	IMethod3(a int, b types_1.S1, c types_1.S2[string], d types_2.S2[*types.Package]) (types_1.S1, error)
}

//genpls:proxy
//genpls:mock
type AliasIface = I1
