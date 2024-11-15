// Package errors provides Wrap in addition to the functions in the stdlib errors package.
// It's otherwise a drop-in replacement for the stdlib package.
package errors

import (
	"errors"
	"fmt"
)

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func New(text string) error {
	return errors.New(text)
}

func Newf(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Wrap(err error, format string, a ...interface{}) error {
	if err == nil {
		return nil
	}
	var fmtArgs []interface{}
	fmtArgs = append(fmtArgs, a...)
	fmtArgs = append(fmtArgs, err)
	return fmt.Errorf(format+": %w", fmtArgs...)
}
