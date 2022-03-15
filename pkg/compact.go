package server

import (
	"fmt"
	"time"
)

const (
	DefaultCompactionInterval  = 1 * time.Minute
	DefaultCompactionRetention = 2
)

func Compaction(tick *time.Ticker, stop <-chan struct{}) {
	for {
		select {
		case <-tick.C:
			appDB.Compaction()
		case <-stop:
			return
		}
	}
}

// Perform a simple retention based compaction
func (s *Store) Compaction() {
	fmt.Printf("Start compaction at %s.\n", time.Now())
	s.Lock()
	defer s.Unlock()

	for k, v := range s.db {
		if len(v) > DefaultCompactionRetention {
			s.db[k] = v[len(v)-DefaultCompactionRetention:]
			fmt.Printf("Compact %s to the length %v.\n", k, len(s.db[k]))
		}
	}
}
