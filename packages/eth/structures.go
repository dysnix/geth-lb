package eth

import "encoding/json"

type Request struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      uint64          `json:"id"`
	Result  json.RawMessage `json:"result"`
}

type Params []string
type RequestsBath []Request
type ResponsesBath []Response
