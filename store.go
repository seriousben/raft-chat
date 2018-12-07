package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"sync"
	"time"

	"go.etcd.io/etcd/etcdserver/api/snap"
)

type Post struct {
	User     string
	Message  string
	PostedAt time.Time
}

type chatStore struct {
	proposeC    chan<- string
	mu          sync.RWMutex
	rooms       map[string][]Post
	snapshotter *snap.Snapshotter
	wsCommitsC  chan<- roomPost
}

type roomPost struct {
	RoomName string
	Post     Post
}

func newChatStore(snapshotter *snap.Snapshotter, proposeC chan<- string, commitC <-chan *string, wsCommitsC chan<- roomPost, errorC <-chan error) *chatStore {
	s := &chatStore{
		proposeC:    proposeC,
		rooms:       make(map[string][]Post),
		snapshotter: snapshotter,
		wsCommitsC:  wsCommitsC,
	}
	// replay log into chatStore map
	s.readCommits(commitC, errorC)
	// read commits from raft into chatStore map until error
	go s.readCommits(commitC, errorC)
	return s
}

func (s *chatStore) Rooms() []string {
	s.mu.RLock()
	keys := make([]string, len(s.rooms))
	i := 0
	for k := range s.rooms {
		keys[i] = k
		i++
	}
	s.mu.RUnlock()
	return keys
}

func (s *chatStore) Lookup(roomName string) ([]Post, bool) {
	s.mu.RLock()
	v, ok := s.rooms[roomName]
	s.mu.RUnlock()
	return v, ok
}

func (s *chatStore) Propose(roomName string, post Post) {
	var buf bytes.Buffer
	post.PostedAt = time.Now()
	if err := gob.NewEncoder(&buf).Encode(roomPost{
		RoomName: roomName,
		Post:     post,
	}); err != nil {
		log.Fatal(err)
	}
	s.proposeC <- buf.String()
}

func (s *chatStore) readCommits(commitC <-chan *string, errorC <-chan error) {
	for data := range commitC {
		if data == nil {
			// done replaying log; new data incoming
			// OR signaled to load snapshot
			snapshot, err := s.snapshotter.Load()
			if err == snap.ErrNoSnapshot {
				return
			}
			if err != nil {
				log.Panic(err)
			}
			log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
			if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
				log.Panic(err)
			}
			continue
		}

		var dataRoomPost roomPost
		dec := gob.NewDecoder(bytes.NewBufferString(*data))
		if err := dec.Decode(&dataRoomPost); err != nil {
			log.Fatalf("raft-chat: could not decode message (%v)", err)
		}
		s.mu.Lock()
		s.rooms[dataRoomPost.RoomName] = append(s.rooms[dataRoomPost.RoomName], dataRoomPost.Post)
		s.mu.Unlock()
		s.wsCommitsC <- dataRoomPost
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}

func (s *chatStore) getSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.rooms)
}

func (s *chatStore) recoverFromSnapshot(snapshot []byte) error {
	var store map[string][]Post
	if err := json.Unmarshal(snapshot, &store); err != nil {
		log.Println("error recover snapshot", err)
		return err
	}
	s.mu.Lock()
	s.rooms = store
	s.mu.Unlock()
	return nil
}
