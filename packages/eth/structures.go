package eth

import "encoding/json"

type Params []string

type Request struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int32           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int32           `json:"id"`
	Result  json.RawMessage `json:"result"`
}
