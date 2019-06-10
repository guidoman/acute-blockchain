package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
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
	QueryLatest MessageType = iota
	QueryAll
	ResponseBlockchain
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

type Message struct {
	Type *MessageType `json:"type,omitempty"`
}

type BlockchainResponseMessage struct {
	Message
	Payload []Block `json:"data",omitempty`
}

var genesisBlock = Block{0, 0, "1465154705", "בראשית", "816534932c2b7154836da6afc367695e6337db8a921823784c14378abed4f7d7"}
var blockchain = []Block{genesisBlock}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var peers = make(map[*websocket.Conn]bool) // TODO process.env.PEERS ? process.env.PEERS.split(',') : [];

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

func getBlocksEndpoint(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(blockchain)
}

func handleMineBlock(hub *Hub, w http.ResponseWriter, r *http.Request) {
	msgData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	newBlock := generateNextBlock(string(msgData))
	addBlock(newBlock)
	hub.broadcast <- responseLatestMsg()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleBroadcast(hub *Hub, w http.ResponseWriter, r *http.Request) {
	msgData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	hub.broadcast <- msgData
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func connectToPeers(hub *Hub, w http.ResponseWriter, r *http.Request) {
	msgData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	u := url.URL{Scheme: "ws", Host: string(msgData), Path: "/peer"}
	log.Printf("connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("Error connecting to peer: ", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()

	client.send <- queryChainLengthMsg()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func queryChainLengthMsg() []byte {
	t := QueryLatest
	msg := Message{&t}
	res, _ := json.Marshal(msg)
	return res
}

func queryAllMsg() []byte {
	t := QueryAll
	msg := Message{&t}
	res, _ := json.Marshal(msg)
	return res
}

func initP2PServer(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()

	client.send <- queryChainLengthMsg()
}

func initHttpServer(hub *Hub, port string) {
	router := mux.NewRouter()

	router.HandleFunc("/blocks", getBlocksEndpoint).Methods("GET")

	router.HandleFunc("/mineBlock", func(w http.ResponseWriter, r *http.Request) {
		handleMineBlock(hub, w, r)
	}).Methods("POST")
	router.HandleFunc("/peer", func(w http.ResponseWriter, r *http.Request) {
		initP2PServer(hub, w, r)
	})

	router.HandleFunc("/addPeer", func(w http.ResponseWriter, r *http.Request) {
		connectToPeers(hub, w, r)
	}).Methods("POST")

	router.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		handleBroadcast(hub, w, r)
	}).Methods("POST")

	router.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		clients := hub.getRegisteredClients()
		msg := struct {
			Peers []string `json:"peers"`
		}{Peers: make([]string, len(clients))}
		for i, c := range clients {
			msg.Peers[i] = fmt.Sprintf("%s", c.conn.RemoteAddr())
		}
		jsonData, _ := json.Marshal(msg)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonData))
	})

	fmt.Printf("Listening http on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msgData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
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
		case QueryLatest:
			fmt.Println("read: QUERY_LATEST")
			c.send <- responseLatestMsg()
		case QueryAll:
			fmt.Println("read: QUERY_ALL")
			c.send <- responseChainMsg()
		case ResponseBlockchain:
			fmt.Println("read: RESPONSE_BLOCKCHAIN")
			handleBlockChainResponse(msgData, c.hub)
		default:
			fmt.Println("Unknown message type: ", *msg.Type)
		}
	}
}

func handleBlockChainResponse(msgData []byte, hub *Hub) {
	var msg BlockchainResponseMessage
	if err := json.Unmarshal(msgData, &msg); err != nil {
		fmt.Println("Error parsing message: ", err)
	}
	receivedBlocks := msg.Payload
	sort.Slice(receivedBlocks, func(i, j int) bool {
		return receivedBlocks[i].Index < receivedBlocks[j].Index
	})
	latestBlockReceived := receivedBlocks[len(receivedBlocks)-1]
	latestBlockHeld := getLatestBlock()
	if latestBlockReceived.Index > latestBlockHeld.Index {
		fmt.Printf("Blockchain possibly behind. We got: %d, peer got: %d", latestBlockHeld.Index, latestBlockReceived.Index)
		if latestBlockHeld.Hash == latestBlockReceived.Hash {
			fmt.Println("We can append the received block to our chain")
			blockchain = append(blockchain, latestBlockReceived)
			hub.broadcast <- responseLatestMsg()
		} else if len(receivedBlocks) == 1 {
			fmt.Println("We have to query the chain from our peer")
			hub.broadcast <- queryAllMsg()
		} else {
			fmt.Println("Received blockchain is longer than current blockchain")
			replaceChain(receivedBlocks, hub)
		}
	} else {
		fmt.Println("Received blockchain is not longer than current blockchain. Do nothing")
	}
}

func replaceChain(newBlocks []Block, hub *Hub) {
	if isValidChain(newBlocks) && len(newBlocks) > len(blockchain) {
		fmt.Println("Received blockchain is valid. Replacing current blockchain with received blockchain")
		blockchain = newBlocks
		hub.broadcast <- responseLatestMsg()
	} else {
		fmt.Println("Received blockchain is invalid")
	}
}

func isValidChain(blockchainToValidate []Block) bool {
	if blockchainToValidate[0] != genesisBlock {
		return false
	}
	tempBlocks := []Block{blockchainToValidate[0]}
	for i := 1; i < len(blockchainToValidate); i++ {
		if res, _ := isValidNewBlock(blockchainToValidate[i], tempBlocks[i-1]); !res {
			return false
		}
		tempBlocks = append(tempBlocks, blockchainToValidate[i])
	}
	return true
}

func responseLatestMsg() []byte {
	t := ResponseBlockchain
	msg := BlockchainResponseMessage{}
	msg.Type = &t
	msg.Payload = []Block{getLatestBlock()}
	res, _ := json.Marshal(msg)
	return res
}

func responseChainMsg() []byte {
	t := ResponseBlockchain
	msg := BlockchainResponseMessage{}
	msg.Type = &t
	msg.Payload = blockchain
	res, _ := json.Marshal(msg)
	return res
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func main() {
	// See: https://github.com/gorilla/websocket/tree/master/examples/chat
	// See: https://rogerwelin.github.io/golang/websockets/gorilla/2018/03/13/golang-websockets.html
	// See: https://www.thepolyglotdeveloper.com/2016/07/create-a-simple-restful-api-with-golang/
	fmt.Println("Starting acute blockchain...")
	hub := newHub()
	go hub.run()
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "3000"
	}
	initHttpServer(hub, port)
}
