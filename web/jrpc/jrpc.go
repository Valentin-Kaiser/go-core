package jrpc

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ContextKey represents keys for context values
type ContextKey string

const (
	// ContextKeyResponseWriter key for http.ResponseWriter
	ContextKeyResponseWriter ContextKey = "response"
	// ContextKeyRequest key for *http.Request
	ContextKeyRequest ContextKey = "request"
	// ContextKeyWebSocketConn key for *websocket.Conn
	ContextKeyWebSocketConn ContextKey = "websocket"
)

// Service implements the RTLS-Suite API server with support for both
// HTTP and WebSocket endpoints. It provides automatic method dispatch,
// protocol buffer message handling, and context enrichment.
type Service struct {
	Server
}

// Server represents a jRPC service implementation.
type Server interface {
	// Descriptor returns the protocol buffer file descriptor for the service.
	Descriptor() protoreflect.FileDescriptor
}

// New creates a new jrpc service instance and registers the provided
// service implementation. The service implementation has to implement the Descriptor method.
func New(s Server) *Service {
	return &Service{Server: s}
}

// Handle processes HTTP POST requests to API endpoints.
// It performs method resolution, request validation, message unmarshaling,
// method invocation, and response marshaling. The context is enriched with
// HTTP components for use by service methods.
//
// URL format: /{service}/{method}
// Content-Type: application/json (Protocol Buffer JSON format)
//
// Parameters:
//   - w: HTTP ResponseWriter for sending the response
//   - r: HTTP Request containing the API call
func (s *Service) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := WithHTTPContext(r.Context(), w, r)

	service := r.PathValue("service")
	method := r.PathValue("method")
	md, err := s.find(service, method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	msg, err := s.message(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.ContentLength > 0 {
		body, _ := io.ReadAll(r.Body)
		if len(body) != int(r.ContentLength) {
			http.Error(w, "body length does not match Content-Length", http.StatusBadRequest)
			return
		}
		err = protojson.Unmarshal(body, msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	defer apperror.Catch(r.Body.Close, "closing request body failed")

	resp, err := s.call(ctx, method, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := s.marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(out)
	if err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

// HandleWebsocket processes WebSocket connections for streaming API endpoints.
// It validates method signatures against protocol buffer definitions, determines
// the streaming pattern (bidirectional, server-side, or client-side), and routes
// to the appropriate handler. The context is enriched with both HTTP and WebSocket
// components for comprehensive access within service methods.
//
// Supported streaming patterns:
//   - Bidirectional: func(ctx, chan *InputMsg, chan OutputMsg) error
//   - Server streaming: func(ctx, *InputMsg, chan OutputMsg) error
//   - Client streaming: func(ctx, chan *InputMsg) (OutputMsg, error)
//
// Parameters:
//   - w: HTTP ResponseWriter from the WebSocket upgrade
//   - r: HTTP Request from the WebSocket upgrade
//   - conn: Established WebSocket connection
func (s *Service) HandleWebsocket(w http.ResponseWriter, r *http.Request, conn *websocket.Conn) {
	service := r.PathValue("service")
	method := r.PathValue("method")

	md, err := s.find(service, method)
	if err != nil {
		s.closeWS(conn, websocket.CloseInternalServerErr, "service or method not found")
		return
	}

	m := reflect.ValueOf(s.Server).MethodByName(method)
	if !m.IsValid() {
		s.closeWS(conn, websocket.CloseInternalServerErr, "method not found")
		return
	}
	mt := m.Type()

	streamingType, err := s.validateMethodSignature(mt, md)
	if err != nil {
		s.closeWS(conn, websocket.CloseInternalServerErr, "invalid method signature: "+err.Error())
		return
	}

	switch streamingType {
	case StreamingTypeBidirectional:
		s.handleBidirectionalStream(r.Context(), conn, m, mt)
	case StreamingTypeServerStream:
		s.handleServerStream(r.Context(), conn, m, mt, md)
	case StreamingTypeClientStream:
		s.handleClientStream(r.Context(), conn, m, mt)
	case StreamingTypeUnary:
		s.closeWS(conn, websocket.CloseInternalServerErr, "unary methods are not supported over WebSocket")
	default:
		s.closeWS(conn, websocket.CloseInternalServerErr, "unsupported streaming type")
	}
}

// WithHTTPContext enriches the provided context with HTTP request components.
// It adds the ResponseWriter and Request to the context, making them available
// throughout the request processing pipeline for logging, middleware, and
// other cross-cutting concerns.
//
// Parameters:
//   - ctx: The base context to enrich
//   - w: The HTTP ResponseWriter for the current request
//   - r: The HTTP Request being processed
//
// Returns the enriched context containing the HTTP components.
func WithHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	ctx = context.WithValue(ctx, ContextKeyResponseWriter, w)
	ctx = context.WithValue(ctx, ContextKeyRequest, r)
	return ctx
}

// WithWebSocketContext adds a WebSocket connection to the context.
// This enables WebSocket-specific operations and connection management
// from within streaming service methods.
//
// Parameters:
//   - ctx: The base context to enrich
//   - conn: The WebSocket connection for the current session
//
// Returns the enriched context containing the WebSocket connection.
func WithWebSocketContext(ctx context.Context, conn *websocket.Conn) context.Context {
	return context.WithValue(ctx, ContextKeyWebSocketConn, conn)
}

// GetResponseWriter extracts the HTTP ResponseWriter from the context.
// This allows service methods to access the original response writer for
// setting custom headers, status codes, or other HTTP-specific operations.
//
// Parameters:
//   - ctx: The context containing the ResponseWriter
//
// Returns:
//   - http.ResponseWriter: The response writer if found
//   - bool: True if the ResponseWriter was found in the context
func GetResponseWriter(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(ContextKeyResponseWriter).(http.ResponseWriter)
	return w, ok
}

// GetRequest extracts the HTTP Request from the context.
// This provides access to request metadata such as headers, URL parameters,
// authentication information, and other request-specific data for logging,
// authorization, and business logic purposes.
//
// Parameters:
//   - ctx: The context containing the HTTP Request
//
// Returns:
//   - *http.Request: The HTTP request if found
//   - bool: True if the Request was found in the context
func GetRequest(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(ContextKeyRequest).(*http.Request)
	return r, ok
}

// GetWebSocketConn extracts the WebSocket connection from the context.
// This enables streaming service methods to access connection properties,
// configure timeouts, handle connection-specific operations, and manage
// the WebSocket lifecycle.
//
// Parameters:
//   - ctx: The context containing the WebSocket connection
//
// Returns:
//   - *websocket.Conn: The WebSocket connection if found
//   - bool: True if the connection was found in the context
func GetWebSocketConn(ctx context.Context) (*websocket.Conn, bool) {
	conn, ok := ctx.Value(ContextKeyWebSocketConn).(*websocket.Conn)
	return conn, ok
}

func (s *Service) call(ctx context.Context, method string, req proto.Message) (any, error) {
	m := reflect.ValueOf(s.Server).MethodByName(method)
	if !m.IsValid() {
		return nil, apperror.NewError("method not found")
	}

	mt := m.Type()
	if mt.NumIn() != 2 || mt.NumOut() != 2 {
		return nil, errors.New("invalid method signature")
	}
	if !mt.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, errors.New("first argument must be context.Context")
	}
	if !mt.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, errors.New("second return value must be error")
	}

	wanted := mt.In(1)
	if wanted.Kind() != reflect.Ptr {
		return nil, errors.New("request must be a pointer")
	}

	reqVal := reflect.ValueOf(req)
	if !reqVal.IsValid() {
		return nil, errors.New("nil request")
	}

	if !reqVal.Type().AssignableTo(wanted) {
		// Convert via JSON round-trip using protojson to the expected type.
		reqPtr := reflect.New(wanted.Elem())
		b, err := protojson.Marshal(req)
		if err != nil {
			return nil, err
		}
		pm, ok := reqPtr.Interface().(proto.Message)
		if !ok {
			return nil, errors.New("expected proto.Message for request")
		}
		if err := protojson.Unmarshal(b, pm); err != nil {
			return nil, err
		}
		reqVal = reqPtr
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reqVal})
	res := outs[0].Interface()
	var err error
	if e := outs[1].Interface(); e != nil {
		err = e.(error)
	}
	return res, err
}

func (s *Service) find(service, method string) (protoreflect.MethodDescriptor, error) {
	sd := s.Descriptor().Services().ByName(protoreflect.Name(service))
	if sd == nil {
		return nil, apperror.NewError("service not found")
	}

	md := sd.Methods().ByName(protoreflect.Name(method))
	if md == nil {
		return nil, apperror.NewError("method not found")
	}
	return md, nil
}

func (s *Service) message(md protoreflect.MethodDescriptor) (proto.Message, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(md.Input().FullName())
	if err != nil {
		log.Error().Err(err).Msg("failed to find message type")
		return dynamicpb.NewMessage(md.Input()), nil
	}

	return mt.New().Interface(), nil
}

func (s *Service) marshal(v any) ([]byte, error) {
	if pm, ok := v.(proto.Message); ok {
		return protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}.Marshal(pm)
	}
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.CanAddr() {
		if pm, ok := rv.Addr().Interface().(proto.Message); ok {
			return protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseProtoNames:   true,
			}.Marshal(pm)
		}
	}
	// Handle non-pointer values by creating a pointer to them
	if rv.IsValid() && rv.Kind() == reflect.Struct {
		// Create a new pointer to the struct type and copy the value
		ptrType := reflect.PointerTo(rv.Type())
		if ptrType.Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			newPtr := reflect.New(rv.Type())
			newPtr.Elem().Set(rv)
			if pm, ok := newPtr.Interface().(proto.Message); ok {
				return protojson.MarshalOptions{
					EmitUnpopulated: true,
					UseProtoNames:   true,
				}.Marshal(pm)
			}
		}
	}
	return nil, errors.New("response is not a proto message")
}

// handleBidirectionalStream handles bidirectional streaming WebSocket connections
func (s *Service) handleBidirectionalStream(ctx context.Context, conn *websocket.Conn, m reflect.Value, mt reflect.Type) {
	inType, outType := mt.In(1), mt.In(2)
	inPtr, outPtr := inType.Elem(), outType.Elem()
	in, out := reflect.MakeChan(inType, 0), reflect.MakeChan(outType, 0)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	read := s.startMessageReader(ctx, conn, in, inPtr)
	write := s.startMessageWriter(ctx, conn, out, outPtr)

	done := make(chan error, 1)
	go func() {
		outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), in, out})
		e := outs[0].Interface()
		if e != nil {
			err, ok := e.(error)
			if !ok {
				log.Error().Msg("method returned non-error type in error position")
			}
			done <- err
		} else {
			done <- nil
		}
		out.Close()
	}()

	var final error
	select {
	case final = <-done:
	case final = <-read:
	}
	<-write

	if final != nil && !websocket.IsCloseError(final, websocket.CloseNormalClosure) {
		s.closeWS(conn, websocket.CloseInternalServerErr, final.Error())
		return
	}
	s.closeWS(conn, websocket.CloseNormalClosure, "")
}

// handleServerStream handles server streaming WebSocket connections
func (s *Service) handleServerStream(ctx context.Context, conn *websocket.Conn, m reflect.Value, mt reflect.Type, md protoreflect.MethodDescriptor) {
	outType := mt.In(2)
	outPtr := outType.Elem()
	out := reflect.MakeChan(outType, 0)

	msg, err := s.message(md)
	if err != nil {
		s.closeWS(conn, websocket.CloseInternalServerErr, "failed to create request message")
		return
	}

	// Read initial message
	reqPtr := reflect.New(mt.In(1).Elem())
	err = s.readWSMessage(conn, reqPtr)
	if err != nil {
		s.closeWS(conn, websocket.CloseInternalServerErr, "failed to read initial message: "+err.Error())
		return
	}

	if reqPtr.IsValid() && len(reflect.ValueOf(msg).Elem().String()) > 0 {
		// Copy from read message to the expected message type
		b, err := protojson.Marshal(reqPtr.Interface().(proto.Message))
		if err != nil {
			s.closeWS(conn, websocket.CloseInternalServerErr, "failed to marshal initial message")
			return
		}
		err = protojson.Unmarshal(b, msg)
		if err != nil {
			s.closeWS(conn, websocket.CloseInternalServerErr, "failed to unmarshal initial message")
			return
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	write := s.startMessageWriter(ctx, conn, out, outPtr)

	wanted := mt.In(1)
	reqVal := reflect.ValueOf(msg)
	if !reqVal.Type().AssignableTo(wanted) {
		s.closeWS(conn, websocket.CloseInternalServerErr, "request message is of wrong type")
		return
	}

	done := make(chan error, 1)
	go func() {
		outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reqVal, out})
		e := outs[0].Interface()
		if e != nil {
			err, ok := e.(error)
			if !ok {
				log.Error().Msg("method returned non-error type in error position")
			}
			done <- err
		} else {
			done <- nil
		}
		out.Close()
	}()

	final := <-done
	<-write

	if final != nil && !websocket.IsCloseError(final, websocket.CloseNormalClosure) {
		s.closeWS(conn, websocket.CloseInternalServerErr, final.Error())
		return
	}
	s.closeWS(conn, websocket.CloseNormalClosure, "")
}

// handleClientStream handles client streaming WebSocket connections
func (s *Service) handleClientStream(ctx context.Context, conn *websocket.Conn, m reflect.Value, mt reflect.Type) {
	inType := mt.In(1)
	inPtr := inType.Elem()
	in := reflect.MakeChan(inType, 0)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	read := s.startMessageReader(ctx, conn, in, inPtr)

	done := make(chan struct {
		resp any
		err  error
	}, 1)
	go func() {
		outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), in})
		var res any = outs[0].Interface()
		e := outs[1].Interface()
		if e != nil {
			err, ok := e.(error)
			if !ok {
				log.Error().Msg("method returned non-error type in error position")
			}
			done <- struct {
				resp any
				err  error
			}{nil, err}
			return
		}
		done <- struct {
			resp any
			err  error
		}{res, nil}
	}()

	var final struct {
		resp any
		err  error
	}
	select {
	case final = <-done:
	case err := <-read:
		final.err = err
	}

	if final.err != nil && !websocket.IsCloseError(final.err, websocket.CloseNormalClosure) {
		s.closeWS(conn, websocket.CloseInternalServerErr, final.err.Error())
		return
	}

	if final.resp != nil {
		err := s.writeWSMessage(conn, reflect.ValueOf(final.resp), reflect.TypeOf(final.resp))
		if err != nil {
			s.closeWS(conn, websocket.CloseInternalServerErr, err.Error())
			return
		}
	}
	s.closeWS(conn, websocket.CloseNormalClosure, "")
}
func (s *Service) readWSMessage(conn *websocket.Conn, msgPtr reflect.Value) error {
	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	if messageType != websocket.TextMessage {
		return errors.New("only text messages are supported")
	}

	if msgPtr.Type().Kind() != reflect.Ptr {
		return errors.New("message type is not a pointer")
	}

	msg, ok := msgPtr.Interface().(proto.Message)
	if !ok {
		return errors.New("message type is not a proto message")
	}

	if len(payload) > 0 {
		err = protojson.Unmarshal(payload, msg)
		if err != nil {
			return err
		}
	}

	return nil
}

// startMessageReader starts a goroutine to read messages from WebSocket into a channel
func (s *Service) startMessageReader(ctx context.Context, conn *websocket.Conn, inChan reflect.Value, inPtr reflect.Type) <-chan error {
	read := make(chan error, 1)
	go func() {
		defer close(read)
		defer inChan.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			reqPtr := reflect.New(inPtr.Elem())
			err := s.readWSMessage(conn, reqPtr)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				read <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				inChan.Send(reqPtr)
			}
		}
	}()
	return read
}

// writeWSMessage marshals and writes a proto message to the WebSocket
func (s *Service) writeWSMessage(conn *websocket.Conn, val reflect.Value, t reflect.Type) error {
	var out any
	switch t.Kind() {
	case reflect.Ptr, reflect.Struct:
		out = val.Interface()
	default:
		return apperror.NewError("unsupported type for websocket message")
	}

	data, err := s.marshal(out)
	if err != nil {
		return apperror.Wrap(err)
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return apperror.Wrap(err)
	}
	return nil
}

// startMessageWriter starts a goroutine to write messages from a channel to WebSocket
func (s *Service) startMessageWriter(ctx context.Context, conn *websocket.Conn, outChan reflect.Value, outPtr reflect.Type) <-chan struct{} {
	write := make(chan struct{})
	go func() {
		defer close(write)
		for {
			val, ok := outChan.Recv()
			if !ok {
				return
			}

			err := s.writeWSMessage(conn, val, outPtr)
			if err != nil {
				log.Error().Err(err).Msg("failed to write websocket message")
				return
			}
		}
	}()
	return write
}

func (s *Service) closeWS(conn *websocket.Conn, code int, reason string) {
	err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	if err != nil && !errors.Is(err, websocket.ErrCloseSent) && !errors.Is(err, net.ErrClosed) {
		log.Error().Err(err).Msg("failed to send websocket close message")
	}
	err = conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Error().Err(err).Msg("failed to close websocket connection")
	}
}

// StreamingType represents the type of streaming for a method
type StreamingType int

const (
	StreamingTypeUnary StreamingType = iota
	StreamingTypeBidirectional
	StreamingTypeServerStream
	StreamingTypeClientStream
	StreamingTypeInvalid
)

// validateMethodSignature validates and determines the streaming type of a method
func (s *Service) validateMethodSignature(mt reflect.Type, md protoreflect.MethodDescriptor) (StreamingType, error) {
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	// Basic validation: must have at least context parameter and error return
	if mt.NumIn() < 1 || !mt.In(0).Implements(contextType) {
		return StreamingTypeInvalid, errors.New("first parameter must be context.Context")
	}
	if mt.NumOut() < 1 || !mt.Out(mt.NumOut()-1).Implements(errorType) {
		return StreamingTypeInvalid, errors.New("last return value must be error")
	}

	// Get expected message types from proto descriptor
	inputType, err := s.getProtoMessageType(md.Input())
	if err != nil {
		return StreamingTypeInvalid, errors.New("failed to resolve input message type: " + err.Error())
	}
	outputType, err := s.getProtoMessageType(md.Output())
	if err != nil {
		return StreamingTypeInvalid, errors.New("failed to resolve output message type: " + err.Error())
	}

	// Determine streaming type based on proto descriptor and validate signature
	isServerStreaming := md.IsStreamingServer()
	isClientStreaming := md.IsStreamingClient()

	switch {
	case isServerStreaming && isClientStreaming:
		// Bidirectional streaming: func(ctx, chan *InputMsg, chan OutputMsg) error
		if mt.NumIn() != 3 || mt.NumOut() != 1 {
			return StreamingTypeInvalid, errors.New("bidirectional streaming method must have signature: func(context.Context, chan *InputMsg, chan OutputMsg) error")
		}

		// Validate input channel type: chan *InputMsg
		if mt.In(1).Kind() != reflect.Chan || mt.In(1).Elem().Kind() != reflect.Ptr {
			return StreamingTypeInvalid, errors.New("second parameter must be chan *InputMsg")
		}
		actualInputType := mt.In(1).Elem().Elem() // chan *T -> T
		if !s.typesMatch(actualInputType, inputType) {
			return StreamingTypeInvalid, errors.New("input channel type mismatch: expected chan *" + inputType.String() + ", got " + mt.In(1).String())
		}

		// Validate output channel type: chan OutputMsg
		if mt.In(2).Kind() != reflect.Chan {
			return StreamingTypeInvalid, errors.New("third parameter must be chan OutputMsg")
		}
		actualOutputType := mt.In(2).Elem() // chan T -> T
		// For output, we need to handle both *T and T cases
		if actualOutputType.Kind() == reflect.Ptr {
			actualOutputType = actualOutputType.Elem()
		}
		if !s.typesMatch(actualOutputType, outputType) {
			return StreamingTypeInvalid, errors.New("output channel type mismatch: expected chan " + outputType.String() + ", got " + mt.In(2).String())
		}

		return StreamingTypeBidirectional, nil

	case isServerStreaming && !isClientStreaming:
		// Server streaming: func(ctx, *InputMsg, chan OutputMsg) error
		if mt.NumIn() != 3 || mt.NumOut() != 1 {
			return StreamingTypeInvalid, errors.New("server streaming method must have signature: func(context.Context, *InputMsg, chan OutputMsg) error")
		}

		// Validate input type: *InputMsg
		if mt.In(1).Kind() != reflect.Ptr {
			return StreamingTypeInvalid, errors.New("second parameter must be *InputMsg")
		}
		actualInputType := mt.In(1).Elem() // *T -> T
		if !s.typesMatch(actualInputType, inputType) {
			return StreamingTypeInvalid, errors.New("input type mismatch: expected *" + inputType.String() + ", got " + mt.In(1).String())
		}

		// Validate output channel type: chan OutputMsg
		if mt.In(2).Kind() != reflect.Chan || mt.In(2).ChanDir()&reflect.SendDir == 0 {
			return StreamingTypeInvalid, errors.New("third parameter must be send chan OutputMsg")
		}
		actualOutputType := mt.In(2).Elem() // chan T -> T
		if actualOutputType.Kind() == reflect.Ptr {
			actualOutputType = actualOutputType.Elem()
		}
		if !s.typesMatch(actualOutputType, outputType) {
			return StreamingTypeInvalid, errors.New("output channel type mismatch: expected chan " + outputType.String() + ", got " + mt.In(2).String())
		}

		return StreamingTypeServerStream, nil

	case !isServerStreaming && isClientStreaming:
		// Client streaming: func(ctx, chan *InputMsg) (OutputMsg, error)
		if mt.NumIn() != 2 || mt.NumOut() != 2 {
			return StreamingTypeInvalid, errors.New("client streaming method must have signature: func(context.Context, chan *InputMsg) (OutputMsg, error)")
		}

		// Validate input channel type: chan *InputMsg
		if mt.In(1).Kind() != reflect.Chan || mt.In(1).Elem().Kind() != reflect.Ptr {
			return StreamingTypeInvalid, errors.New("second parameter must be chan *InputMsg")
		}
		actualInputType := mt.In(1).Elem().Elem() // chan *T -> T
		if !s.typesMatch(actualInputType, inputType) {
			return StreamingTypeInvalid, errors.New("input channel type mismatch: expected chan *" + inputType.String() + ", got " + mt.In(1).String())
		}

		// Validate output type: OutputMsg or *OutputMsg
		actualOutputType := mt.Out(0)
		if actualOutputType.Kind() == reflect.Ptr {
			actualOutputType = actualOutputType.Elem()
		}
		if !s.typesMatch(actualOutputType, outputType) {
			return StreamingTypeInvalid, errors.New("output type mismatch: expected " + outputType.String() + ", got " + mt.Out(0).String())
		}

		return StreamingTypeClientStream, nil

	default:
		// Unary: func(ctx, *InputMsg) (OutputMsg, error)
		if mt.NumIn() != 2 || mt.NumOut() != 2 {
			return StreamingTypeInvalid, errors.New("unary method must have signature: func(context.Context, *InputMsg) (OutputMsg, error)")
		}

		// Validate input type: *InputMsg
		if mt.In(1).Kind() != reflect.Ptr {
			return StreamingTypeInvalid, errors.New("second parameter must be *InputMsg")
		}
		actualInputType := mt.In(1).Elem() // *T -> T
		if !s.typesMatch(actualInputType, inputType) {
			return StreamingTypeInvalid, errors.New("input type mismatch: expected *" + inputType.String() + ", got " + mt.In(1).String())
		}

		// Validate output type: OutputMsg or *OutputMsg
		actualOutputType := mt.Out(0)
		if actualOutputType.Kind() == reflect.Ptr {
			actualOutputType = actualOutputType.Elem()
		}
		if !s.typesMatch(actualOutputType, outputType) {
			return StreamingTypeInvalid, errors.New("output type mismatch: expected " + outputType.String() + ", got " + mt.Out(0).String())
		}

		return StreamingTypeUnary, nil
	}
}

// getProtoMessageType resolves a proto message descriptor to its Go reflect.Type
func (s *Service) getProtoMessageType(msgDesc protoreflect.MessageDescriptor) (reflect.Type, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(msgDesc.FullName())
	if err != nil {
		// Fallback to dynamic message
		dynMsg := dynamicpb.NewMessage(msgDesc)
		return reflect.TypeOf(dynMsg).Elem(), nil
	}
	return reflect.TypeOf(mt.New().Interface()).Elem(), nil
}

// typesMatch checks if two reflect.Types represent the same proto message type
func (s *Service) typesMatch(actual, expected reflect.Type) bool {
	// Direct type comparison
	if actual == expected {
		return true
	}

	// Compare by type name as fallback
	actualName := actual.String()
	expectedName := expected.String()

	// Handle package differences - compare the base type name
	if actualName == expectedName {
		return true
	}

	// Extract the struct name for comparison
	actualParts := actual.Name()
	expectedParts := expected.Name()

	return actualParts == expectedParts
}
