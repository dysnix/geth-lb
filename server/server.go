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
	var reqs eth.RequestsBath
	var resp eth.Response
	var resps eth.ResponsesBath

	// Dump request for debug
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	// Parse request
	defer r.Body.Close()
	requestBody, err := ioutil.ReadAll(r.Body)

	errDecode := json.Unmarshal(requestBody, &req)
	if errDecode != nil {
		errDecode = json.Unmarshal(requestBody, &reqs)
		if errDecode != nil {
			log.Printf("Error: %s. Request %s", errDecode, string(requestDump))
			http.Error(w, errDecode.Error(), 500)
			return
		} else {
			for _, element := range reqs {
				// Proxy request to Geth backend
				resp = eth.RpcCall(element)

				// Handle and modify JsonRPC response
				resp = eth.HandleResponse(req, resp)

				resps = append(resps, resp)
			}
			respsData, err := json.Marshal(resps)
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(w, bytes.NewBuffer(respsData))
		}
	} else {
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
