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
	"os"
	"strconv"
	"strings"
	"time"
)

const RedisDatabaseDefault = "0"
const RawTxCacheExpireTime = 60 * 60 * 24 * 30

var BackendUrl string
var RedisAddress string
var RedisDatabase string
var RedisClient *redis.Client

func init() {
	BackendUrl = GetEnvOrDefault("BACKEND_URL", "")
	RedisAddress = GetEnvOrDefault("REDIS_ADDRESS", "")
	RedisDatabase = GetEnvOrDefault("REDIS_DATABASE", RedisDatabaseDefault)
	RedisDatabaseInt, err := strconv.ParseInt(RedisDatabase, 10, 64)
	if err != nil {
		panic(err)
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: RedisAddress,
		DB:   int(RedisDatabaseInt), // use default DB
	})
}

func GetEnvOrDefault(key string, defValue string) string {
	var value string

	value = os.Getenv(key)

	if value == "" && defValue == "" {
		log.Panicf("Please set env variable \"%s\"", key)
	}

	if value == "" {
		return defValue
	}

	return value
}

func toRpcResult(value string) []byte {
	return []byte(fmt.Sprintf("\"%s\"", value))
}

func fromRpcResult(value []byte) string {
	return strings.Replace(string(value), "\"", "", -1)
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

func GetTransactionCount(address string, origResult string) string {
	// Own method for overwrite
	// JsonRPC method GetTransactionCount
	var val string
	var localValue uint64
	var origValue uint64

	origValue, err := hexutil.DecodeUint64(origResult)
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
	paramsData, _ := json.Marshal(Params{address, "latest"})

	var req = Request{
		JsonRpc: "jsonrpc",
		Id:      1,
		Method:  "eth_getTransactionCount",
		Params:  paramsData,
	}
	resp := RpcCall(req)

	result, err := hexutil.DecodeUint64(fromRpcResult(resp.Result))
	if err != nil {
		panic(err)
	}

	return result
}

func isRawTxExist(rawTx string) bool {
	val, _ := RedisClient.Get(rawTx).Result()
	if val == "" {
		_, err := RedisClient.Set(rawTx, "exist", time.Second*RawTxCacheExpireTime).Result()
		if err != nil {
			log.Panicln(err)
		}
		return false
	} else {
		return true
	}
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

	if !isRawTxExist(rawTx) {
		_, err = RedisClient.Incr(senderAddress).Result()
		if err != nil {
			panic(err)
		}
	}
}

func HandleResponse(req Request, resp Response) Response {
	var params Params

	switch method := req.Method; method {
	case "eth_getTransactionCount":
		// Overwrite getTransactionCount result
		json.NewDecoder(bytes.NewBuffer(req.Params)).Decode(&params)
		if params[1] == "latest" {
			resp.Result = toRpcResult(GetTransactionCount(params[0], fromRpcResult(resp.Result)))
		}
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
