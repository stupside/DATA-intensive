package relay

import (
	"context"
	"fmt"
	"sync"

	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// session represents an active relay session coordinating producer and consumer
type session struct {
	// Context for this session
	ctx context.Context
	// Cancel function to stop the session
	cancel context.CancelFunc

	// relayID is the unique identifier for this relay session
	relayID bson.ObjectID

	// channel to send chunk requests from consumer to producer
	requestChan chan *v1.RequestChunk
	consumeChan chan *v1.ConsumeChunk
}

// newSession creates a new relay session
func newSession(parent context.Context, bufferSize int) *session {
	// Derive a cancelable session context from the provided parent
	ctx, cancel := context.WithCancel(parent)

	return &session{
		ctx:         ctx,
		cancel:      cancel,
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

// SessionStore manages active relay sessions in memory.
// Sessions are ephemeral and exist only during active file transfers.
// For persistent relay history, see MongoDB repositories (RelayRepository, ChunkRepository).
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[bson.ObjectID]*session
}

// NewSessionStore creates a new in-memory session store for relay coordination
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[bson.ObjectID]*session),
	}
}

// Create creates a new in-memory session for relay coordination
func (s *SessionStore) Create(parent context.Context, relayID bson.ObjectID, bufferSize int) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[relayID]; exists {
		return nil, errors.InvalidArgumentMsg(parent, fmt.Sprintf("Session already exists for relay %s", relayID.Hex()))
	}

	// Create in-memory session
	sess := newSession(parent, bufferSize)
	s.sessions[relayID] = sess

	return sess, nil
}

// Get retrieves a session by relay ID
func (s *SessionStore) Get(relayID bson.ObjectID) (*session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[relayID]
	return session, exists
}

// Remove deletes session from in-memory store
func (s *SessionStore) Remove(ctx context.Context, relayID bson.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[relayID]
	if exists {
		session.close()
	}

	delete(s.sessions, relayID)

	return nil
}
