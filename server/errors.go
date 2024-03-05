package server

import "fmt"

const (
	// DefaultErrorCode default error code
	DefaultErrorCode = -32000
	// InvalidRequestErrorCode error code for invalid requests
	InvalidRequestErrorCode = -32600
	// NotFoundErrorCode error code for not found objects
	NotFoundErrorCode = -32601
	// InvalidParamsErrorCode error code for invalid parameters
	InvalidParamsErrorCode = -32602
	// ParserErrorCode error code for parsing errors
	ParserErrorCode = -32700
)

var (
	// ErrBatchRequestsDisabled returned by the pool mananger server when a batch request is detected and the batch requests are disabled via configuration
	ErrBatchRequestsDisabled = fmt.Errorf("batch requests are disabled")
	// ErrBatchRequestsLimitExceeded returned by the server when a batch request is detected and the number of requests are greater than the configured limit
	ErrBatchRequestsLimitExceeded = fmt.Errorf("batch requests limit exceeded")
)

// Error interface
type Error interface {
	Error() string
	ErrorCode() int
	ErrorData() []byte
}

// ServerError represents an error returned by a pool manager endpoint
type ServerError struct {
	err  string
	code int
	data []byte
}

// NewServerError creates a new error instance to be returned by the pool manager endpoints
func NewServerError(code int, err string, args ...interface{}) *ServerError {
	return NewServerErrorWithData(code, err, nil, args...)
}

// NewServerErrorWithData creates a new error instance with data to be returned by the pool manager endpoints
func NewServerErrorWithData(code int, err string, data []byte, args ...interface{}) *ServerError {
	var errMessage string
	if len(args) > 0 {
		errMessage = fmt.Sprintf(err, args...)
	} else {
		errMessage = err
	}
	return &ServerError{code: code, err: errMessage, data: data}
}

// Error returns the error message
func (e ServerError) Error() string {
	return e.err
}

// ErrorCode returns the error code
func (e *ServerError) ErrorCode() int {
	return e.code
}

// ErrorData returns the error data
func (e *ServerError) ErrorData() []byte {
	return e.data
}
