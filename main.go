package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Block is the main block Data type
type Block struct {
	Index        int64  `json:"index"`
	Timestamp    int64  `json:"timestamp"`
	PreviousHash string `json:"previousHas"`
	Data         string `json:"data"`
	Hash         string `json:"hash"`
}

type MessageType int

const (
	QUERY_LATEST MessageType = iota
	QUERY_ALL
	RESPONSE_BLOCKCHAIN
)

type Message struct {
	Type *MessageType `json:"type,omitempty"`
}

type BlockchainResponseMessage struct {
	Message
	Blockchain []Block `json:"blockchain",omitempty`
}

var genesisBlock = Block{0, 0, "1465154705", "my genesis block!!", "816534932c2b7154836da6afc367695e6337db8a921823784c14378abed4f7d7"}
var blockchain = []Block{genesisBlock}
var broadcast = make(chan []byte)

var upgrader = websocket.Upgrader{
	// TODO check the following
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var peers = make(map[*websocket.Conn]bool) // TODO process.env.PEERS ? process.env.PEERS.split(',') : [];

// ComputeHash computes the Hash of a block struct
func computeHashForBlock(block Block) string {
	return computeHash(block.Index, block.Timestamp, block.PreviousHash, block.Data)
}

func computeHash(index int64, timestamp int64, previousHash string, data string) string {
	s := fmt.Sprintf("%d%d%s%s", index, timestamp, previousHash, data)
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getLatestBlock() Block {
	return blockchain[len(blockchain)-1]
}

func isValidNewBlock(newBlock Block, previousBlock Block) (bool, error) {
	if previousBlock.Index+1 != newBlock.Index {
		return false, fmt.Errorf("invalid Index")
	} else if previousBlock.Hash != newBlock.PreviousHash {
		return false, fmt.Errorf("invalid PreviousHash")
	} else if computeHashForBlock(newBlock) != newBlock.Hash {
		return false, fmt.Errorf("invalid Hash")
	}
	return true, nil
}

func addBlock(block Block) error {
	if ok, err := isValidNewBlock(block, getLatestBlock()); !ok {
		return err
	}
	blockchain = append(blockchain, block)
	return nil
}

func getCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

func generateNextBlock(blockData string) Block {
	previousBlock := getLatestBlock()
	nextIndex := previousBlock.Index + 1
	nextTimestamp := getCurrentTimestamp()
	nextHash := computeHash(nextIndex, nextTimestamp, previousBlock.Hash, blockData)
	return Block{nextIndex, nextTimestamp, previousBlock.Hash, blockData, nextHash}
}

func getBlocksEndpoint(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode(blockchain)
}

func broadcastHandler(w http.ResponseWriter, r *http.Request) {
	responseData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	go writer(responseData)
}

func peerHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	// register peer
	peers[conn] = true
	conn.WriteMessage(websocket.TextMessage, []byte(queryChainLengthMsg()))

	//
	defer conn.Close()
	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		fmt.Println("recv: ", string(msgData))
		var msg Message
		err = json.Unmarshal(msgData, &msg)
		if err != nil {
			fmt.Println("Error parsing message: ", err)
			continue
		}
		if msg.Type == nil {
			fmt.Println("Empty message type")
			continue
		}
		switch *msg.Type {
		case QUERY_LATEST:
			fmt.Println("QUERY_LATEST")
			// TODO handle message
		case QUERY_ALL:
			fmt.Println("QUERY_ALL")
			// TODO handle message
		case RESPONSE_BLOCKCHAIN:
			fmt.Println("RESPONSE_BLOCKCHAIN")
			// TODO handle message
		default:
			fmt.Println("Unknown message type: ", *msg.Type)
		}
	}
}

func queryChainLengthMsg() string {
	return "{\"type\": 0}" // TODO
}

func writer(message []byte) {
	broadcast <- message
}

func sender() {
	for {
		message := <-broadcast
		// send to every client that is currently connected
		// TODO use a mutex
		for peer := range peers {
			err := peer.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Websocket error: %s", err)
				peer.Close()
				delete(peers, peer)
			}
		}
	}
}

func initHttpServer() {
	router := mux.NewRouter()
	router.HandleFunc("/blocks", getBlocksEndpoint).Methods("GET")
	// TODO
	// See: https://www.thepolyglotdeveloper.com/2016/07/create-a-simple-restful-api-with-golang/

	router.HandleFunc("/peer", peerHandler) // websocker
	router.HandleFunc("/broadcast", broadcastHandler).Methods("POST")

	fmt.Printf("Listening http on port %d", 3000)
	log.Fatal(http.ListenAndServe(":3000", router))
}

func initP2PServer() {
	fmt.Println("Initing P2P...")
	go sender()
}

func main() {
	fmt.Println("Starting acute blockchain...")
	initP2PServer()
	initHttpServer()
	// TODO websockets peers
	// See: https://rogerwelin.github.io/golang/websockets/gorilla/2018/03/13/golang-websockets.html
}
