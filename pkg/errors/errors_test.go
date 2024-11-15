package errors_test

import (
	"testing"

	"github.com/LasseHels/landlord/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func TestWrapReturnsWrappedError(t *testing.T) {
	err := errors.Wrap(errors.New("oops"), "wrap this")
	assert.EqualError(t, err, "wrap this: oops")

	unwrappedErr := errors.Unwrap(err)
	assert.EqualError(t, unwrappedErr, "oops")
}

func TestWrapCanUseFormatParameters(t *testing.T) {
	err := errors.Wrap(errors.New("whoops"), "error in %v", 123)
	assert.EqualError(t, err, "error in 123: whoops")
}

func TestNewfFormatsAnError(t *testing.T) {
	err := errors.Newf("oops, error code %v", 100)
	assert.EqualError(t, err, "oops, error code 100")
}
