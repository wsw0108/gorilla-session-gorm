package gorm

import "time"

type storeOption func(s *Store)

func WithTableName(tableName string) storeOption {
	return func(s *Store) {
		s.tableName = tableName
	}
}

func WithGCInterval(gcInterval time.Duration) storeOption {
	return func(s *Store) {
		s.gcInterval = gcInterval
	}
}

func WithGCDisabled() storeOption {
	return func(s *Store) {
		s.gcDisabled = true
	}
}

func WithInitTableDisabled() storeOption {
	return func(s *Store) {
		s.initTableDisabled = true
	}
}

func WithSecureDisabled() storeOption {
	return func(s *Store) {
		s.secureDisabled = true
	}
}

func WithKeyPairs(keyPairs ...[]byte) storeOption {
	return func(s *Store) {
		s.keyPairs = keyPairs
	}
}
