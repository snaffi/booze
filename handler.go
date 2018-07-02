package booze

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
)

var (
	HandlerAlreadyExist = errors.New("handler already exist")
)

type RPCHandlerFunc func(ctx context.Context, payload *Payload) (interface{}, *Error)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type RPCHandler struct {
	methods map[string]RPCHandlerFunc
}

func NewRPCHandler() *RPCHandler {
	return &RPCHandler{
		methods: make(map[string]RPCHandlerFunc),
	}
}

func (h *RPCHandler) Register(methodName string, handlerFunc RPCHandlerFunc) {
	_, exist := h.methods[methodName]
	if exist {
		panic(HandlerAlreadyExist)
	}
	h.methods[methodName] = handlerFunc
}

func (h *RPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	responseChan := make(chan interface{}, 128)
	defer conn.Close()

	go h.writeResponses(responseChan, conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected connection error: %v", err)
			}
			return
		}
		//if batch request
		if bytes.HasPrefix(message, []byte("[")) {
			requests := make([][]byte, 0)
			_, err = jsonparser.ArrayEach(message, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				requests = append(requests, value)
			})
			if err != nil {
				resp := Response{Error: &ParseError}
				resp.Error.Data = err.Error()
				responseChan <- &resp
				continue
			}
			go func(requests [][]byte) {
				responses := make([]*Response, len(requests))
				wg := sync.WaitGroup{}
				for i := 0; i < len(requests); i++ {
					wg.Add(1)
					go func(requestNum int) {
						responses[requestNum] = h.handle(r.Context(), requests[requestNum])
						wg.Done()
					}(i)
				}
				wg.Wait()
				responseChan <- responses
			}(requests)
		} else {
			go func() {
				responseChan <- h.handle(r.Context(), message)
			}()
		}
	}
}

func (h *RPCHandler) handle(ctx context.Context, data []byte) *Response {
	var resp Response
	var payload Payload

	err := json.Unmarshal(data, &payload)
	if err != nil {
		switch err.(type) {
		case *json.UnmarshalTypeError:
			resp.Error = &InvalidParams
		case *json.SyntaxError, *json.InvalidUnmarshalError:
			resp.Error = &ParseError
		default:
			resp.Error = &InternalError
		}
		resp.Error.Data = err.Error()
		return &resp
	}

	rpcHandler, exist := h.methods[payload.Method]
	if !exist {
		resp.ID = payload.ID
		resp.Error = &MethodNotFound
		return &resp
	}

	result, handlerError := rpcHandler(ctx, &payload)
	resp.ID = payload.ID
	if handlerError != nil {
		resp.Error = handlerError
	} else {
		resp.Result = result
	}

	return &resp
}

func (h *RPCHandler) writeResponses(responses chan interface{}, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case response, ok := <-responses:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := conn.WriteJSON(response)
			if err != nil {
				log.Printf("can not write response: %s\n", err)
				return
			}

			n := len(responses)
			for i := 0; i < n; i++ {
				err := conn.WriteJSON(<-responses)
				if err != nil {
					log.Printf("can not write response: %s\n", err)
					return
				}
			}

		}
	}
}
