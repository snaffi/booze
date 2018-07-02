# booze 🍻
Golang Websocket JSONRPC 2.0 implementation 

## Usage

```go
import (
	"context"
	"log"
	"net/http"
	"github.com/snaffi/booze"
)


type SomeStruct struct {
	Message string `json:"message"`
}

func someRpcHandler(ctx context.Context, payload *booze.Payload) (interface{}, *booze.Error) {
	var object SomeStruct
	err := payload.Unmarshal(&object)
	if err != nil {
		e := &InvalidRequest
		e.Data = err
		return nil, e
	}
	return &object, nil
}

func main() {
	rpcHandler := booze.NewRPCHandler()

	rpcHandler.Register("some_handler", someRpcHandler)

	http.Handle("/ws_rpc", rpcHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

Input data format

```javascript

{
    "id": "1",
    "method": "some_handler",
    "params": {
        "message": "hi"
    }
}

```

Output data format

```javascript

{
    "id": "1",
    "result": {
        "message": "hi"
    },
    "jsonrpc":"2.0"
}

```

