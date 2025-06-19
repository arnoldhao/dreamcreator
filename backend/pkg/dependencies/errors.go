package dependencies

import "fmt"

// 定义错误类型
type ErrorType string

const (
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeDownload   ErrorType = "download"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypePermission ErrorType = "permission"
)

// DependencyError 依赖错误
type DependencyError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *DependencyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *DependencyError) Unwrap() error {
	return e.Cause
}

// 错误构造函数
func NewNotFoundError(message string, cause error) *DependencyError {
	return &DependencyError{Type: ErrorTypeNotFound, Message: message, Cause: cause}
}

func NewDownloadError(message string, cause error) *DependencyError {
	return &DependencyError{Type: ErrorTypeDownload, Message: message, Cause: cause}
}
