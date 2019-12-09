package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"geth-lb/packages/eth"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
)

const ListenPortDefault = "8545"

var ListenPort string

func handler(w http.ResponseWriter, r *http.Request) {
	var req eth.Request
	var resp eth.Response

	// Parse request
	errDecode := json.NewDecoder(r.Body).Decode(&req)
	if errDecode != nil {
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(requestDump))
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s", b)

		http.Error(w, errDecode.Error(), 400)
		return
	}

	// Proxy request to Geth backend
	resp = eth.RpcCall(req)

	// Handle and modify JsonRPC response
	resp = eth.HandleResponse(req, resp)

	respData, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(w, bytes.NewBuffer(respData))
}

func main() {
	ListenPort = eth.GetEnvOrDefault("LISTEN_PORT", ListenPortDefault)
	ListenPortInt, err := strconv.ParseInt(ListenPort, 10, 64)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handler)

	log.Printf("geth-lb starts on proxy %d", ListenPortInt)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", ListenPortInt), nil))
}
