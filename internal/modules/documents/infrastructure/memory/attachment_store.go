package memory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

type AttachmentStore struct {
	mu    sync.RWMutex
	blobs map[string][]byte
}

func NewAttachmentStore() *AttachmentStore {
	return &AttachmentStore{blobs: map[string][]byte{}}
}

func (s *AttachmentStore) Save(_ context.Context, storageKey string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blobs[storageKey] = append([]byte(nil), content...)
	return nil
}

func (s *AttachmentStore) Open(_ context.Context, storageKey string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.blobs[storageKey]
	if !ok {
		return nil, fmt.Errorf("attachment blob not found")
	}
	content = append([]byte(nil), content...)
	return io.NopCloser(bytes.NewReader(content)), nil
}

func (s *AttachmentStore) Delete(_ context.Context, storageKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blobs, storageKey)
	return nil
}
