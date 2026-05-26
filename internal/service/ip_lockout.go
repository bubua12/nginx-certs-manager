package service

import (
	"sync"
	"time"
)

const (
	maxFailedAttempts = 5
	lockDuration      = 30 * time.Minute
)

type ipRecord struct {
	Attempts  int
	LockedAt  time.Time
	IsLocked  bool
}

type IPLockout struct {
	mu      sync.RWMutex
	records map[string]*ipRecord
}

func NewIPLockout() *IPLockout {
	l := &IPLockout{
		records: make(map[string]*ipRecord),
	}
	go l.cleanup()
	return l
}

func (l *IPLockout) Check(ip string) (locked bool, remaining int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rec, ok := l.records[ip]
	if !ok {
		return false, maxFailedAttempts
	}

	if rec.IsLocked {
		if time.Since(rec.LockedAt) >= lockDuration {
			return false, maxFailedAttempts
		}
		return true, 0
	}

	return false, maxFailedAttempts - rec.Attempts
}

func (l *IPLockout) RecordFailure(ip string) (locked bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.records[ip]
	if !ok {
		rec = &ipRecord{}
		l.records[ip] = rec
	}

	if rec.IsLocked {
		return true
	}

	rec.Attempts++
	if rec.Attempts >= maxFailedAttempts {
		rec.IsLocked = true
		rec.LockedAt = time.Now()
		return true
	}
	return false
}

func (l *IPLockout) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, ip)
}

func (l *IPLockout) GetLockRemaining(ip string) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rec, ok := l.records[ip]
	if !ok || !rec.IsLocked {
		return 0
	}

	elapsed := time.Since(rec.LockedAt)
	if elapsed >= lockDuration {
		return 0
	}
	return lockDuration - elapsed
}

func (l *IPLockout) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, rec := range l.records {
			if rec.IsLocked && time.Since(rec.LockedAt) >= lockDuration {
				delete(l.records, ip)
			} else if !rec.IsLocked && time.Since(rec.LockedAt) >= lockDuration {
				delete(l.records, ip)
			}
		}
		l.mu.Unlock()
	}
}
