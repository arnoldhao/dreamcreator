package dependencies

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDependencyError(t *testing.T) {
	causeErr := errors.New("original error")
	depErr := &DependencyError{
		Type:    ErrorTypeDownload,
		Message: "download failed",
		Cause:   causeErr,
	}

	// 测试 Error() 方法
	errorMsg := depErr.Error()
	assert.Contains(t, errorMsg, "download")
	assert.Contains(t, errorMsg, "download failed")
	assert.Contains(t, errorMsg, "original error")

	// 测试 Unwrap() 方法
	unwrapped := depErr.Unwrap()
	assert.Equal(t, causeErr, unwrapped)
}

func TestDependencyErrorWithoutCause(t *testing.T) {
	depErr := &DependencyError{
		Type:    ErrorTypeNotFound,
		Message: "dependency not found",
		Cause:   nil,
	}

	errorMsg := depErr.Error()
	assert.Equal(t, "not_found: dependency not found", errorMsg)

	unwrapped := depErr.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestErrorConstructors(t *testing.T) {
	causeErr := errors.New("cause")

	// 测试 NewNotFoundError
	notFoundErr := NewNotFoundError("not found", causeErr)
	assert.Equal(t, ErrorTypeNotFound, notFoundErr.Type)
	assert.Equal(t, "not found", notFoundErr.Message)
	assert.Equal(t, causeErr, notFoundErr.Cause)

	// 测试 NewDownloadError
	downloadErr := NewDownloadError("download failed", causeErr)
	assert.Equal(t, ErrorTypeDownload, downloadErr.Type)
	assert.Equal(t, "download failed", downloadErr.Message)
	assert.Equal(t, causeErr, downloadErr.Cause)
}

func TestErrorTypes(t *testing.T) {
	assert.Equal(t, ErrorType("not_found"), ErrorTypeNotFound)
	assert.Equal(t, ErrorType("download"), ErrorTypeDownload)
	assert.Equal(t, ErrorType("validation"), ErrorTypeValidation)
	assert.Equal(t, ErrorType("permission"), ErrorTypePermission)
}
