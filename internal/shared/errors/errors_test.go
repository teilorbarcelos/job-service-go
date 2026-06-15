package errors

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	e := New("X", "boom", 500)
	assert.Equal(t, "boom", e.Error())
}

func TestAppError_WithCause(t *testing.T) {
	cause := stderrors.New("root")
	e := Wrap("X", "wrapped", 500, cause)
	assert.Equal(t, "wrapped: root", e.Error())
	assert.Same(t, cause, e.Unwrap())
}

func TestNewConfigurationError(t *testing.T) {
	e := NewConfigurationError("missing env")
	assert.Equal(t, "CONFIGURATION_ERROR", e.Code)
	assert.Equal(t, 500, e.StatusCode)
}

func TestNewValidationError(t *testing.T) {
	e := NewValidationError("bad input")
	assert.Equal(t, "VALIDATION_ERROR", e.Code)
	assert.Equal(t, 400, e.StatusCode)
}

func TestNewConnectionError(t *testing.T) {
	e := NewConnectionError("RabbitMQ", "timeout")
	assert.Equal(t, "CONNECTION_ERROR", e.Code)
	assert.Equal(t, 503, e.StatusCode)
	assert.Contains(t, e.Message, "RabbitMQ")
	assert.Contains(t, e.Message, "timeout")
}

func TestIsAppError_True(t *testing.T) {
	var err error = NewConfigurationError("x")
	assert.True(t, IsAppError(err))
}

func TestIsAppError_False(t *testing.T) {
	assert.False(t, IsAppError(stderrors.New("plain")))
}
