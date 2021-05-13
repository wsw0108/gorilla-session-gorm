package gorm

import "time"

type StoreOption func(s *Store)

func WithTableName(tableName string) StoreOption {
	return func(s *Store) {
		s.tableName = tableName
	}
}

func WithGCInterval(gcInterval time.Duration) StoreOption {
	return func(s *Store) {
		s.gcInterval = gcInterval
	}
}

func WithGCDisabled() StoreOption {
	return func(s *Store) {
		s.gcDisabled = true
	}
}

func WithInitTableDisabled() StoreOption {
	return func(s *Store) {
		s.initTableDisabled = true
	}
}

// WithSecureDisabled encode session values as JSON, ONLY for development
func WithSecureDisabled() StoreOption {
	return func(s *Store) {
		s.secureDisabled = true
	}
}

func WithKeyPairs(keyPairs ...[]byte) StoreOption {
	return func(s *Store) {
		s.keyPairs = keyPairs
	}
}
