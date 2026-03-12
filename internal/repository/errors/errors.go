package repository

import "errors"

var (
	ErrUnsupportedType = errors.New("the metric type unsupport this action")
	ErrFieldUndefine   = errors.New("required field is not define")
	ErrNotFound        = errors.New("notfound")
)
