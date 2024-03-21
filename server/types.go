package server

import "encoding/json"

// Request is a jsonrpc Request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a jsonrpc success/error response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Id      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject is a jsonrpc error
type ErrorObject struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    *ArgBytes `json:"data,omitempty"`
}

// ArgBytes helps to marshal byte array values provided in the RPC requests
type ArgBytes []byte

// ArgBytesPtr helps to marshal byte array values provided in the RPC requests
func ArgBytesPtr(b []byte) *ArgBytes {
	bb := ArgBytes(b)

	return &bb
}

// NewResponse returns Success/Error response object
func NewResponse(req Request, reply []byte, err Error) Response {
	var result json.RawMessage
	if reply != nil {
		result = reply
	}

	var errorObj *ErrorObject
	if err != nil {
		errorObj = &ErrorObject{
			Code:    err.ErrorCode(),
			Message: err.Error(),
		}
		if err.ErrorData() != nil {
			errorObj.Data = ArgBytesPtr(err.ErrorData())
		}
	}

	return Response{
		JSONRPC: req.JSONRPC,
		Id:      req.ID,
		Result:  result,
		Error:   errorObj,
	}
}
