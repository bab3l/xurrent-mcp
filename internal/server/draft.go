package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// Draft represents a pending mutation that requires user approval.
type Draft struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
	Summary   string          `json:"summary"`
	Method    string          `json:"method"`
	Entity    string          `json:"entity"`
	Path      string          `json:"path"`
	Body      json.RawMessage `json:"body,omitempty"`
}

// DraftStore holds pending drafts in memory with TTL expiration.
type DraftStore struct {
	mu    sync.RWMutex
	drafts map[string]*Draft
	ttl   time.Duration
}

// NewDraftStore creates a draft store with the given TTL.
func NewDraftStore(ttl time.Duration) *DraftStore {
	ds := &DraftStore{
		drafts: make(map[string]*Draft),
		ttl:    ttl,
	}
	go ds.cleanupLoop()
	return ds
}

// Create returns a new draft with a random ID.
func (ds *DraftStore) Create(summary, method, entity, path string, body json.RawMessage) *Draft {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	d := &Draft{
		ID:        newDraftID(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ds.ttl),
		Summary:   summary,
		Method:    method,
		Entity:    entity,
		Path:      path,
		Body:      body,
	}
	ds.drafts[d.ID] = d
	return d
}

// Get retrieves a draft by ID. Returns nil if not found or expired.
func (ds *DraftStore) Get(id string) *Draft {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	d, ok := ds.drafts[id]
	if !ok || time.Now().After(d.ExpiresAt) {
		return nil
	}
	return d
}

// Remove deletes a draft by ID.
func (ds *DraftStore) Remove(id string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.drafts, id)
}

// List returns all non-expired drafts.
func (ds *DraftStore) List() []*Draft {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	now := time.Now()
	var out []*Draft
	for _, d := range ds.drafts {
		if now.Before(d.ExpiresAt) {
			out = append(out, d)
		}
	}
	return out
}

func (ds *DraftStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ds.purgeExpired()
	}
}

func (ds *DraftStore) purgeExpired() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	now := time.Now()
	for id, d := range ds.drafts {
		if now.After(d.ExpiresAt) {
			delete(ds.drafts, id)
		}
	}
}

func newDraftID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "draft-" + hex.EncodeToString(b)
}
