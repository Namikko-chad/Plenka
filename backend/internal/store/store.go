// Package store is the server's own key/value storage: one bbolt file (plenka.db) with a
// bucket per feature and JSON-encoded values. The torrent engine keeps its own config.db.
package store

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Store wraps a bbolt database.
type Store struct {
	db *bolt.DB
}

// Open opens (or creates) <dir>/plenka.db.
func Open(dir string) (*Store, error) {
	db, err := bolt.Open(filepath.Join(dir, "plenka.db"), 0o600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database file.
func (s *Store) Close() error { return s.db.Close() }

// Get decodes the value at bucket/key into v. It reports false when the key is absent.
func (s *Store) Get(bucket, key string, v any) (bool, error) {
	var raw []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if val := b.Get([]byte(key)); val != nil {
			raw = append([]byte(nil), val...)
		}
		return nil
	})
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, v)
}

// Put stores v as JSON at bucket/key, creating the bucket if needed.
func (s *Store) Put(bucket, key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), raw)
	})
}

// Delete removes bucket/key. Missing keys are not an error.
func (s *Store) Delete(bucket, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
}

// ErrStop can be returned from a ForEach callback to stop iterating without an error.
var ErrStop = errors.New("stop")

// ForEach calls fn with every key and raw JSON value in the bucket, in key order.
func (s *Store) ForEach(bucket string, fn func(key string, raw []byte) error) error {
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error { return fn(string(k), v) })
	})
	if errors.Is(err, ErrStop) {
		return nil
	}
	return err
}
