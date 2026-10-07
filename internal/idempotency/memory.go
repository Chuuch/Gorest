package idempotency

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory Store for tests.
type MemoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]Record)}
}

func memoryKey(organizationID uuid.UUID, key string) string {
	return organizationID.String() + ":" + key
}

func (s *MemoryStore) TryClaim(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
	method string,
	path string,
	requestHash string,
	expiresAt time.Time,
) (*Record, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	id := memoryKey(organizationID, key)
	if existing, ok := s.records[id]; ok {
		if time.Now().UTC().After(existing.ExpiresAt) {
			delete(s.records, id)
		} else {
			if existing.RequestHash != requestHash || existing.Method != method || existing.Path != path {
				return nil, ErrKeyMismatch
			}
			if !existing.Complete() {
				return nil, ErrKeyInProgress
			}
			copy := existing
			return &copy, nil
		}
	}

	s.records[id] = Record{
		OrganizationID: organizationID,
		Key:            key,
		Method:         method,
		Path:           path,
		RequestHash:    requestHash,
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      expiresAt,
	}
	return nil, nil
}

func (s *MemoryStore) Complete(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
	statusCode int,
	responseBody []byte,
) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	id := memoryKey(organizationID, key)
	rec, ok := s.records[id]
	if !ok {
		return ErrKeyInProgress
	}
	rec.StatusCode = &statusCode
	rec.ResponseBody = append([]byte(nil), responseBody...)
	s.records[id] = rec
	return nil
}

func (s *MemoryStore) Release(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, memoryKey(organizationID, key))
	return nil
}
