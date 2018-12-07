package main

import (
	"flag"
	"strings"

	"go.etcd.io/etcd/raft/raftpb"
)

func main() {
	cluster := flag.String("cluster", "http://127.0.0.1:9021", "comma separated cluster peers")
	id := flag.Int("id", 1, "node ID")
	chatPort := flag.Int("port", 9121, "chat server port")
	join := flag.Bool("join", false, "join an existing cluster")
	flag.Parse()

	proposeC := make(chan string)
	defer close(proposeC)
	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)
	wsCommitsC := make(chan roomPost, 5)
	defer close(wsCommitsC)

	// raft provides a commit stream for the proposals from the http api
	var store *chatStore
	getSnapshot := func() ([]byte, error) { return store.getSnapshot() }
	commitC, errorC, snapshotterReady := newRaftNode(*id, strings.Split(*cluster, ","), *join, getSnapshot, proposeC, confChangeC)

	store = newChatStore(<-snapshotterReady, proposeC, commitC, wsCommitsC, errorC)

	// the chat http handler will propose updates to raft
	serveHTTPChatAPI(store, *chatPort, wsCommitsC, confChangeC, errorC)
}
