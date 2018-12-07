package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.etcd.io/etcd/raft/raftpb"
)

type httpChatAPI struct {
	store       *chatStore
	confChangeC chan<- raftpb.ConfChange
	wsCommitsC  <-chan roomPost
	mu          sync.RWMutex
	wsConns     []*websocket.Conn
}

func checkOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

func (h *httpChatAPI) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade (%v)\n", err)
		http.Error(w, "Failed on upgrade", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.wsConns = append(h.wsConns, conn)
	h.mu.Unlock()
}

func (h *httpChatAPI) readWSCommits() {
	for rp := range h.wsCommitsC {
		h.mu.RLock()
		for _, conn := range h.wsConns {
			conn.WriteJSON(rp)
		}
		h.mu.RUnlock()
	}
}

func (h *httpChatAPI) RoomsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("servings list of rooms")
	rooms := h.store.Rooms()
	enc := json.NewEncoder(w)
	enc.Encode(rooms)
}

func (h *httpChatAPI) RoomMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomName := vars["roomID"]

	if r.Method == "GET" {
		log.Println("servings list of messages in ", roomName)
		room, roomFound := h.store.Lookup(roomName)
		if roomFound {
			enc := json.NewEncoder(w)
			enc.Encode(room)
		} else {
			enc := json.NewEncoder(w)
			enc.Encode([]string{})
		}
	} else {
		log.Println("adding message in ", roomName)
		var post Post
		enc := json.NewDecoder(r.Body)
		err := enc.Decode(&post)

		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		h.store.Propose(roomName, post)
		w.WriteHeader(http.StatusCreated)
	}
}

func (h *httpChatAPI) DefaultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["nodeID"]
	switch {
	case r.Method == "POST":
		url, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		nodeID, err := strconv.ParseUint(nodeName, 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:    raftpb.ConfChangeAddNode,
			NodeID:  nodeID,
			Context: url,
		}
		h.confChangeC <- cc

		// As above, optimistic that raft will apply the conf change
		w.WriteHeader(http.StatusNoContent)
	case r.Method == "DELETE":
		nodeID, err := strconv.ParseUint(nodeName, 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on DELETE", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:   raftpb.ConfChangeRemoveNode,
			NodeID: nodeID,
		}
		h.confChangeC <- cc

		// As above, optimistic that raft will apply the conf change
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "POST")
		w.Header().Add("Allow", "DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func serveHTTPChatAPI(store *chatStore, port int, wsCommitsC <-chan roomPost, confChangeC chan<- raftpb.ConfChange, errorC <-chan error) {
	api := &httpChatAPI{
		store:       store,
		confChangeC: confChangeC,
		wsCommitsC:  wsCommitsC,
		wsConns:     []*websocket.Conn{},
	}

	go api.readWSCommits()

	r := mux.NewRouter()
	r.HandleFunc("/rooms", api.RoomsHandler)
	r.HandleFunc("/rooms/{roomID}", api.RoomMessagesHandler)
	r.HandleFunc("/ws", api.WebsocketHandler)
	r.HandleFunc("/raft/{nodeID}", api.DefaultHandler)

	srv := http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handlers.CORS()(r),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// exit when raft goes down
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}
