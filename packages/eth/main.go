package eth

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/go-redis/redis"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const BackendUrl = "https://rpc-staging.public.test.k8s.2key.net"

var RedisClient *redis.Client

func init() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func GetSenderAddress(rawTx string) string {
	// Decode Raw Transaction for get
	// Sender Address

	var tx *types.Transaction

	// Remove "0x" prefix if exist
	rawTx = strings.Replace(rawTx, "0x", "", -1)

	rawtx, _ := hex.DecodeString(rawTx)
	err := rlp.DecodeBytes(rawtx, &tx)

	if err != nil {
		log.Fatal(err)
	}

	msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId()))
	if err != nil {
		log.Fatal(err)
	}

	// Return sender address
	return msg.From().Hex()
}

func setLocalTxCount(address string, value uint64) {
	_, err := RedisClient.Set(address, fmt.Sprint(value), 0).Result()
	if err != nil {
		panic(err)
	}
}

func parseUint(value string) uint64 {
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		panic(err)
	}
	return result
}

func GetTransactionCount(address string, origResult []byte) string {
	// Own method for overwrite
	// JsonRPC method GetTransactionCount
	var val string
	var localValue uint64
	var origValue uint64

	origValue, err := hexutil.DecodeUint64(string(origResult))
	if err != nil {
		panic(err)
	}

	val, err = RedisClient.Get(address).Result()
	if err != nil {
		// If Redis key not found
		setLocalTxCount(address, origValue)
		return hexutil.EncodeUint64(origValue)
	} else {
		localValue = parseUint(val)
		if origValue > localValue {
			setLocalTxCount(address, origValue)
			return hexutil.EncodeUint64(origValue)
		}
	}

	return hexutil.EncodeUint64(localValue)
}

func RpcCall(req Request) Response {
	var resp Response

	reqData, err := json.Marshal(req)
	log.Println(string(reqData))
	proxyResponseData, err := http.Post(BackendUrl, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		log.Panic(err)
	}

	defer proxyResponseData.Body.Close()

	// Pare response
	err = json.NewDecoder(proxyResponseData.Body).Decode(&resp)
	if err != nil {
		log.Panicf("Error decode response: %s", err)
	}

	return resp
}

func rpcGetTransactionCount(address string) uint64 {
	var req = Request{
		JsonRpc: "jsonrpc",
		Id:      1,
		Method:  "eth_getTransactionCount",
		Params:  []byte(fmt.Sprintf("[\"%s\"]", address)),
	}
	resp := RpcCall(req)

	result, err := hexutil.DecodeUint64(strings.Replace(string(resp.Result), "\"", "", -1))
	if err != nil {
		panic(err)
	}

	return result
}

func SendRawTransaction(rawTx string) {
	// Own method for handle SendRawTransaction
	// JsonRPC method and increment local counter

	senderAddress := GetSenderAddress(rawTx)
	log.Printf("Sender address: %s", senderAddress)

	origValue := rpcGetTransactionCount(senderAddress)

	val, err := RedisClient.Get(senderAddress).Result()
	if err == nil && origValue > parseUint(val) {
		setLocalTxCount(senderAddress, origValue)
	}

	_, err = RedisClient.Incr(senderAddress).Result()
	if err != nil {
		panic(err)
	}
}

func HandleResponse(req Request, resp Response) Response {
	var result []byte
	var params Params

	switch method := req.Method; method {
	case "eth_getTransactionCount":
		// Overwrite getTransactionCount result
		json.NewDecoder(bytes.NewBuffer(req.Params)).Decode(&params)
		result = []byte(fmt.Sprintf("\"%s\"", GetTransactionCount(params[0], []byte(strings.Replace(string(resp.Result), "\"", "", -1)))))
		resp.Result = result
	case "eth_sendRawTransaction":
		// Handle sendRawTransaction request for
		// increment local transactions counter
		json.NewDecoder(bytes.NewBuffer(req.Params)).Decode(&params)
		SendRawTransaction(params[0])
	}

	// Debug logging
	reqData, _ := json.Marshal(resp)
	respData, _ := json.Marshal(resp)
	log.Printf("> Request: %s", string(reqData))
	log.Printf("< Response: %s", string(respData))

	return resp
}
