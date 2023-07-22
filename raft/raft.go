package raft

import (
	"context"
	"math/rand"
	"sync"
	"time"

	pb "github.com/bvulak/tinykv/proto"
)

type State int

const (
	Follower State = iota + 1
	Candidate
	Leader
)

type Raft struct {
	mu           sync.Mutex
	state        State
	self         uint64
	peers        map[uint64]pb.RaftClient
	currentTerm  uint64
	prevLogIndex uint64
	lastLogIndex uint64
	lastLogTerm  uint64
	votedFor     uint64

	heartbeatTimer *time.Timer
	electionTimer  *time.Timer
}

func Start() {
	rand.Seed(time.Now().UnixNano())
	rf := &Raft{
		heartbeatTimer: time.NewTimer(time.Millisecond * time.Duration(rand.Intn(100)+200)),
		electionTimer:  time.NewTimer(time.Millisecond * time.Duration(rand.Intn(100)+200)),
	}
	select {
	case <-rf.electionTimer.C:
		rf.mu.Lock()
		rf.currentTerm++

		rf.electionTimer.Reset(time.Millisecond * time.Duration(rand.Intn(100)+200))
		rf.state = Candidate
		rf.mu.Unlock()
	case <-rf.heartbeatTimer.C:
		rf.mu.Lock()
		rf.mu.Unlock()
	}
}

func (rf *Raft) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (resp *pb.RequestVoteResponse, err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.currentTerm > req.Term ||
		(rf.currentTerm == req.Term && rf.votedFor != req.CandidateID && rf.votedFor != 0) {
		resp.Term, resp.VoteGranted = rf.currentTerm, false
		return
	}
	if rf.currentTerm < req.Term {
		rf.state = Follower
		rf.currentTerm = req.Term
		rf.votedFor = 0
	}
	if rf.lastLogIndex > req.LastLogIndex ||
		(rf.lastLogIndex == req.LastLogIndex && rf.lastLogTerm > req.LastLogTerm) {
		resp.Term, resp.VoteGranted = rf.currentTerm, false
		return
	}
	rf.votedFor = req.CandidateID
	resp.Term, resp.VoteGranted = rf.currentTerm, true
	return
}

func (rf *Raft) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (resp *pb.AppendEntriesResponse, err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.currentTerm > req.Term {
		resp.Term, resp.Success = rf.currentTerm, false
		return
	}
	if rf.currentTerm < req.Term {
		rf.currentTerm = req.Term
		rf.votedFor = 0
	}
	rf.state = Follower
	if rf.prevLogIndex > req.PrevLogIndex {
		resp.Term, resp.Success = 0, false
	}
	if rf.prevLogIndex < req.PrevLogIndex {
		switch {
		case rf.lastLogIndex < req.PrevLogIndex:
			resp.Term, resp.Success = rf.currentTerm, false
			resp.ConflictTerm, resp.ConflictIndex = 0, rf.lastLogIndex+1
		case rf.lastLogTerm == req.PrevLogTerm && rf.lastLogIndex == req.PrevLogIndex:

		}

	}

	resp.Term, resp.Success = rf.currentTerm, true
	return
}
