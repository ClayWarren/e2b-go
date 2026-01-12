package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

type (
	// Method is a JSON-RPC method.
	Method string

	// Request is a JSON-RPC request.
	Request struct {
		JSONRPC string `json:"jsonrpc"` // JSONRPC is the JSON-RPC version of the request.
		Method  Method `json:"method"`  // Method is the request method.
		ID      int    `json:"id"`      // ID of the request.
		Params  []any  `json:"params"`  // Params of the request.
	}

	// Response is a JSON-RPC response.
	Response[T any, Q any] struct {
		ID     int `json:"id"`     // ID of the response.
		Result T   `json:"result"` // Result of the response.
		Error  Q   `json:"error"`  // Error of the message.
	}

	// APIError is the error of the API.
	APIError struct {
		Code    int    `json:"code,omitempty"` // Code is the code of the error.
		Message string `json:"message"`        // Message is the message of the error.
	}
)

const (
	rpc = "2.0"
)

func (s *Sandbox) newRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (s *Sandbox) sendRequest(req *http.Request, v interface{}) error {
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < http.StatusOK ||
		res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("failed to create sandbox: %s", res.Status)
	}
	if v == nil {
		return nil
	}
	switch o := v.(type) {
	case *string:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		*o = string(b)
		return nil
	default:
		return json.NewDecoder(res.Body).Decode(v)
	}
}

func decodeResponse[T any, Q any](body []byte) (*Response[T, Q], error) {
	decResp := new(Response[T, Q])
	err := json.Unmarshal(body, decResp)
	if err != nil {
		return nil, err
	}
	return decResp, nil
}

func (s *Sandbox) read(ctx context.Context) error {
	var body []byte
	var err error
	type decResp struct {
		Method string `json:"method"`
		ID     int    `json:"id"`
		Params struct {
			Subscription string `json:"subscription"`
		}
	}
	defer func() {
		err := s.ws.Close()
		if err != nil {
			s.logger.Error("failed to close sandbox", "error", err)
		}
	}()
	msgCh := make(chan []byte, 10)
	for {
		select {
		case body = <-msgCh:
			var decResp decResp
			err = json.Unmarshal(body, &decResp)
			if err != nil {
				return err
			}
			var key any
			key = decResp.Params.Subscription
			if decResp.ID != 0 {
				key = decResp.ID
			}
			toR, ok := s.Map.Load(key)
			if !ok {
				msgCh <- body
				continue
			}
			toRCh, ok := toR.(chan []byte)
			if !ok {
				msgCh <- body
				continue
			}
			s.logger.Debug("read",
				"subscription", decResp.Params.Subscription,
				"body", body,
				"sandbox", s.ID,
			)
			toRCh <- body
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, msg, err := s.ws.ReadMessage()
			if err != nil {
				return err
			}
			msgCh <- msg
		}
	}
}

func (s *Sandbox) writeRequest(
	ctx context.Context,
	method Method,
	params []any,
	respCh chan []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case id := <-s.idCh:
		req := Request{
			Method:  method,
			JSONRPC: rpc,
			Params:  params,
			ID:      id,
		}
		s.logger.Debug("request",
			"sandbox", id,
			"method", method,
			"id", id,
			"params", params,
		)
		s.Map.Store(req.ID, respCh)
		jsVal, err := json.Marshal(req)
		if err != nil {
			return err
		}
		err = s.ws.WriteMessage(websocket.TextMessage, jsVal)
		if err != nil {
			return fmt.Errorf(
				"writing %s request failed (%d): %w",
				method,
				req.ID,
				err,
			)
		}
		return nil
	}
}

func (s *Sandbox) identify(ctx context.Context) {
	id := 1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s.idCh <- id
			id++
		}
	}
}
