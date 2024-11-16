package opt

type Result[T any] struct {
	Ok  T
	Err error
}

func Ok[T any](v T) Result[T] {
	return Result[T]{
		Ok:  v,
		Err: nil,
	}
}

func Err[T any](err error) Result[T] {
	//nolint:exhaustruct
	return Result[T]{
		Err: err,
	}
}
