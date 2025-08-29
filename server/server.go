package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/db"
	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
	"github.com/didip/tollbooth/v6"
)

const (
	wsBufferSizeLimitInBytes = 1024
	maxRequestContentLength  = 1024 * 1024 * 5
	contentType              = "application/json"
)

// https://www.jsonrpc.org/historical/json-rpc-over-http.html#http-header
var acceptedContentTypes = []string{contentType, "application/json-rpc", "application/jsonrequest"}

// Server represents a JSON-RPC server to handle pool-manager requests
type Server struct {
	config     Config
	handler    *Handler
	httpServer *http.Server
	sender     senderInterface
}

// NewServer returns a JSON-RPC server to handle pool-manager requests
func NewServer(cfg Config, poolDB *db.PoolDB, sender senderInterface) *Server {
	endpoints := NewEndpoints(cfg, poolDB, sender)

	handler := newJSONRpcHandler()
	handler.registerEndpoints(endpoints)

	return &Server{config: cfg, handler: handler, sender: sender}
}

// Start initializes pool-manager JSON-RPC server to listen for requests
func (s *Server) Start() {
	if s.httpServer != nil {
		log.Fatalf("HTTP server already started")
	}

	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to create TCP listener, error: %v", err)
	}

	mux := http.NewServeMux()

	lmt := tollbooth.NewLimiter(s.config.MaxRequestsPerIPAndSecond, nil)
	mux.Handle("/", tollbooth.LimitFuncHandler(lmt, s.handle))

	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: s.config.ReadTimeout.Duration,
		ReadTimeout:       s.config.ReadTimeout.Duration,
		WriteTimeout:      s.config.WriteTimeout.Duration,
	}
	log.Infof("HTTP server started at %s", address)
	if err := s.httpServer.Serve(lis); err != nil {
		if err == http.ErrServerClosed {
			log.Fatalf("HTTP server stopped")
		}
		log.Fatalf("closed HTTP connection, error: %v", err)
	}
}

// Stop shutdown the JSON-RPC server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			return err
		}

		if err := s.httpServer.Close(); err != nil {
			return err
		}
		s.httpServer = nil
	}

	return nil
}

func (s *Server) handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if req.Method == http.MethodOptions {
		return
	}

	if req.Method == http.MethodGet {
		_, err := w.Write([]byte("zkEVM Pool Manager"))
		if err != nil {
			log.Error(err)
		}
		return
	}

	if code, err := validateRequest(req); err != nil {
		handleInvalidRequest(w, err, code)
		return
	}

	body := io.LimitReader(req.Body, maxRequestContentLength)
	data, err := io.ReadAll(body)
	if err != nil {
		handleError(w, err)
		return
	}

	single, err := s.isSingleRequest(data)
	if err != nil {
		handleInvalidRequest(w, err, http.StatusBadRequest)
		return
	}

	start := time.Now()
	var respLen int
	if single {
		respLen = s.handleSingleRequest(req, w, data)
	} else {
		respLen = s.handleBatchRequest(req, w, data)
	}
	s.combinedLog(req, start, http.StatusOK, respLen)
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.
func validateRequest(req *http.Request) (int, error) {
	if req.Method != http.MethodPost {
		err := errors.New("method " + req.Method + " not allowed")
		return http.StatusMethodNotAllowed, err
	}

	if req.ContentLength > maxRequestContentLength {
		err := fmt.Errorf("content length too large (%d > %d)", req.ContentLength, maxRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}

	// Check content-type
	if mt, _, err := mime.ParseMediaType(req.Header.Get("content-type")); err == nil {
		for _, accepted := range acceptedContentTypes {
			if accepted == mt {
				return 0, nil
			}
		}
	}
	// Invalid content-type
	err := fmt.Errorf("invalid content type, only %s is supported", contentType)
	return http.StatusUnsupportedMediaType, err
}

func (s *Server) isSingleRequest(data []byte) (bool, error) {
	x := bytes.TrimLeft(data, " \t\r\n")

	if len(x) == 0 {
		return false, fmt.Errorf("empty request body")
	}

	return x[0] != '[', nil
}

func (s *Server) handleSingleRequest(httpRequest *http.Request, w http.ResponseWriter, data []byte) int {
	request, err := s.parseRequest(data)
	if err != nil {
		handleInvalidRequest(w, err, http.StatusBadRequest)
		return 0
	}
	req := handleRequest{Request: request, HttpRequest: httpRequest}
	response := s.handler.Handle(req)

	respBytes, err := json.Marshal(response)
	if err != nil {
		handleError(w, err)
		return 0
	}

	_, err = w.Write(respBytes)
	if err != nil {
		handleError(w, err)
		return 0
	}
	return len(respBytes)
}

func (s *Server) handleBatchRequest(httpRequest *http.Request, w http.ResponseWriter, data []byte) int {
	// Checking if batch requests are enabled
	if !s.config.BatchRequestsEnabled {
		handleInvalidRequest(w, ErrBatchRequestsDisabled, http.StatusBadRequest)
		return 0
	}

	requests, err := s.parseRequests(data)
	if err != nil {
		handleInvalidRequest(w, err, http.StatusBadRequest)
		return 0
	}

	// Checking if batch requests limit is exceeded
	if s.config.BatchRequestsLimit > 0 {
		if len(requests) > int(s.config.BatchRequestsLimit) {
			handleInvalidRequest(w, ErrBatchRequestsLimitExceeded, http.StatusRequestEntityTooLarge)
			return 0
		}
	}

	responses := make([]Response, 0, len(requests))

	for _, request := range requests {
		req := handleRequest{Request: request, HttpRequest: httpRequest}
		response := s.handler.Handle(req)
		responses = append(responses, response)
	}

	respBytes, _ := json.Marshal(responses)
	_, err = w.Write(respBytes)
	if err != nil {
		log.Error(err)
		return 0
	}
	return len(respBytes)
}

func (s *Server) parseRequest(data []byte) (Request, error) {
	var req Request

	if err := json.Unmarshal(data, &req); err != nil {
		return Request{}, fmt.Errorf("invalid json object request body")
	}

	return req, nil
}

func (s *Server) parseRequests(data []byte) ([]Request, error) {
	var requests []Request

	if err := json.Unmarshal(data, &requests); err != nil {
		return nil, fmt.Errorf("invalid json array request body")
	}

	return requests, nil
}

func handleInvalidRequest(w http.ResponseWriter, err error, code int) {
	log.Debugf("invalid request, error: %v", err.Error())
	http.Error(w, err.Error(), code)
}

func handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, syscall.EPIPE) {
		// if it is a broken pipe error, return
		return
	}

	// if it is a different error, write it to the response
	log.Errorf("error processing request, error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// RPCErrorResponse formats error to be returned through RPC
func RPCErrorResponse(code int, message string, err error, logError bool) (interface{}, Error) {
	return RPCErrorResponseWithData(code, message, nil, err, logError)
}

// RPCErrorResponseWithData formats error to be returned through RPC
func RPCErrorResponseWithData(code int, message string, data []byte, err error, logError bool) (interface{}, Error) {
	if logError {
		if err != nil {
			log.Debugf("%v: %v", message, err.Error())
		} else {
			log.Debug(message)
		}
	}
	return nil, NewServerErrorWithData(code, message, data)
}

func (s *Server) combinedLog(r *http.Request, start time.Time, httpStatus, dataLen int) {
	if !s.config.EnableHttpLog {
		return
	}

	log.Infof("%s - - %s \"%s %s %s\" %d %d \"%s\" \"%s\"",
		r.RemoteAddr,
		start.Format("[02/Jan/2006:15:04:05 -0700]"),
		r.Method,
		r.URL.Path,
		r.Proto,
		httpStatus,
		dataLen,
		r.Host,
		r.UserAgent(),
	)
}
