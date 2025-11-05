package relay

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"

	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
)

// session represents an active relay session coordinating producer and consumer
type session struct {
	// relayID is the unique identifier for this relay session
	relayID bson.ObjectID

	// requestChan is used to send chunk requests from consumer to producer
	requestChan chan *v1.RequestChunk

	// consumeChan is used to send chunk data from producer to consumer
	consumeChan chan *v1.ConsumeChunk

	// Context for this session
	ctx context.Context

	// Cancel function to stop the session
	cancel context.CancelFunc
}

// newSession creates a new relay session
func newSession(parent context.Context, relayID bson.ObjectID, bufferSize int) *session {
	// Derive a cancelable session context from the provided parent
	ctx, cancel := context.WithCancel(parent)

	return &session{
		ctx:         ctx,
		cancel:      cancel,
		relayID:     relayID,
		requestChan: make(chan *v1.RequestChunk, bufferSize),
		consumeChan: make(chan *v1.ConsumeChunk, bufferSize),
	}
}

// close closes the session by canceling its context
// Safe to call multiple times due to context.CancelFunc being idempotent
func (s *session) close() {
	s.cancel()
}

// Done returns a channel that's closed when the session is complete
func (s *session) Done() <-chan struct{} {
	return s.ctx.Done()
}

// SessionStore provides thread-safe storage for relay sessions
type SessionStore struct {
	sessions map[bson.ObjectID]*session
	mu       sync.RWMutex
}

// NewSessionStore creates a new session store
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[bson.ObjectID]*session),
	}
}

// Create creates and stores a new session
func (s *SessionStore) Create(parent context.Context, relayID bson.ObjectID, bufferSize int) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[relayID]; exists {
		return nil, fmt.Errorf("session already exists for relay %s", relayID.Hex())
	}

	session := newSession(parent, relayID, bufferSize)
	s.sessions[relayID] = session

	return session, nil
}

// Get retrieves a session by relay ID
func (s *SessionStore) Get(relayID bson.ObjectID) (*session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[relayID]
	return session, exists
}

// Remove removes a session from the store
func (s *SessionStore) Remove(relayID bson.ObjectID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[relayID]
	if exists {
		session.close()
	}

	delete(s.sessions, relayID)
}
