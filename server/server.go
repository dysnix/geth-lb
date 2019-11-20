package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"geth-lb/packages/eth"
	"io"
	"log"
	"net/http"
)

const ListenPort = 8545

func handler(w http.ResponseWriter, r *http.Request) {
	var req eth.Request
	var resp eth.Response

	// Parse request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decode request: %s", err)
		http.Error(w, err.Error(), 400)
		return
	}

	// Proxy request to Geth backend
	resp = eth.RpcCall(req)

	// Handle and modify JsonRPC response
	resp = eth.HandleResponse(req, resp)

	respData, err := json.Marshal(resp)
	io.Copy(w, bytes.NewBuffer(respData))
}

func main() {
	http.HandleFunc("/", handler)

	log.Printf("geth-lb starts on proxy %d", ListenPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", ListenPort), nil))
}
