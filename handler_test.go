package booze

import (
	"context"
	"net/http/httptest"
	"testing"

	"strings"

	"bytes"

	"github.com/gorilla/websocket"
)

func dummySuccessHandler(ctx context.Context, payload *Payload) (interface{}, *Error) {
	return map[string]string{"message": "success"}, nil
}

func dummyErrorHandler(ctx context.Context, payload *Payload) (interface{}, *Error) {
	err := &InvalidRequest
	err.Data = "data"
	return nil, err
}

func TestRPCHandler(t *testing.T) {
	h := NewRPCHandler()
	h.Register("test_ok", dummySuccessHandler)
	h.Register("test_error", dummyErrorHandler)
	ts := httptest.NewServer(h)
	defer ts.Close()
	port := strings.Split(ts.URL, ":")[2]
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:"+port, nil)
	defer conn.Close()

	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		input, output []byte
		name          string
	}{
		{
			[]byte(`{"method": "test_ok"}`),
			[]byte(`{"id":"","result":{"message":"success"},"jsonrpc":"2.0"}`),
			`valid single request`,
		},
		{
			[]byte(`{"method": "test_error"}`),
			[]byte(`{"id":"","error":{"code":-32600,"data":"data","message":"invalid request"},"jsonrpc":"2.0"}`),
			`valid single request with err response`,
		},
		{
			[]byte(`{"method": "unknown_method"}`),
			[]byte(`{"id":"","error":{"code":-32601,"message":"method not found"},"jsonrpc":"2.0"}`),
			`call unknown method`,
		},
		{
			[]byte(`[{"method": "test_ok", "id": "1"}, {"method": "test_ok", "id": "2"}, {"method": "test_ok", "id": "3"}]`),
			[]byte(`[{"id":"1","result":{"message":"success"},"jsonrpc":"2.0"},{"id":"2","result":{"message":"success"},"jsonrpc":"2.0"},{"id":"3","result":{"message":"success"},"jsonrpc":"2.0"}]`),
			`valid multiple requests`,
		},
		{
			[]byte(`[{"method": "test_ok", "id": "1"}, {"method": "test_error", "id": "2"}, {"method": "test_ok", "id": "3"}]`),
			[]byte(`[{"id":"1","result":{"message":"success"},"jsonrpc":"2.0"},{"id":"2","error":{"code":-32600,"data":"data","message":"invalid request"},"jsonrpc":"2.0"},{"id":"3","result":{"message":"success"},"jsonrpc":"2.0"}]`),
			`valid multiple requests with err response`,
		},
		{
			[]byte(`{asd`),
			[]byte(`{"id":"","error":{"code":-32700,"data":"invalid character 'a' looking for beginning of object key string","message":"parse error"},"jsonrpc":"2.0"}`),
			`invalid json`,
		},
		{
			[]byte(`[{"method": "test_ok", "id": "1"}, {"method": "test_ok", "id": "2"}`),
			[]byte(`{"id":"","error":{"code":-32700,"data":"Value is array, but can't find closing ']' symbol","message":"parse error"},"jsonrpc":"2.0"}`),
			`invalid array json`,
		},
		{
			[]byte(`{"method": 12331212}`),
			[]byte(`{"id":"","error":{"code":-32602,"data":"json: cannot unmarshal number into Go struct field Payload.method of type string","message":"invalid params"},"jsonrpc":"2.0"}`),
			`invalid json attrs`,
		},
	}
	for _, c := range cases {
		err = conn.WriteMessage(websocket.TextMessage, []byte(c.input))
		if err != nil {
			t.Fatal(err)
		}
		_, resp, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}

		resp = bytes.TrimSpace(resp)
		if !bytes.Equal(resp, c.output) {
			t.Fatalf("got unexpected response:\n%s\n\nexpected:\n%s\non testcase %s", resp, c.output, c.name)
		}
	}

	drainResponseChannel := struct {
		input [][]byte
		name          string
	}{
		[][]byte{
			[]byte(`{"method": "test_ok"}`),
			[]byte(`{"method": "unknown_method"}`),
			[]byte(`{"method": "test_error"}`),
		},

		`valid single request`,
	}

	for _, input := range drainResponseChannel.input {
		err = conn.WriteMessage(websocket.TextMessage, []byte(input))
		if err != nil {
			t.Fatal(err)
		}
	}

	responsesCount := 0
	for range drainResponseChannel.input {
		_, _, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		responsesCount++
	}
	if responsesCount != len(drainResponseChannel.input) {
		t.Fatalf("got unexpected responses count: %d, expected: %d", responsesCount, len(drainResponseChannel.input))
	}

}
