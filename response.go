package booze

type Error struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}

func (e Error) Error() string {
	return e.Message
}

var (
	ParseError       = Error{Code: -32700, Message: "parse error"}
	SystemError      = Error{Code: -32400, Message: "system error"}
	InternalError    = Error{Code: -32603, Message: "internal error"}
	InvalidParams    = Error{Code: -32602, Message: "invalid params"}
	TransportError   = Error{Code: -32300, Message: "transport error"}
	InvalidRequest   = Error{Code: -32600, Message: "invalid request"}
	MethodNotFound   = Error{Code: -32601, Message: "method not found"}
	ApplicationError = Error{Code: -32500, Message: "application error"}
)

type RPCVersion20 string

func (v RPCVersion20) MarshalJSON() ([]byte, error) {
	return []byte(`"2.0"`), nil
}

type Response struct {
	ID         string       `json:"id"`
	Error      *Error       `json:"error,omitempty"`
	Result     interface{}  `json:"result,omitempty"`
	RPCVersion RPCVersion20 `json:"jsonrpc"`
}
