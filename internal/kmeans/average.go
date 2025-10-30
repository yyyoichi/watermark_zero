package kmeans

import "sync"

type AverageStore struct {
	sum   float64
	count int
	mu    sync.Mutex
}

func (s *AverageStore) Add(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sum += value
	s.count += 1
}

func (s *AverageStore) Average() float64 { return s.sum / float64(s.count) }

func (s *AverageStore) Count() int { return s.count }

func (s *AverageStore) Sum() float64 { return s.sum }

func (s *AverageStore) Initialize(sum float64, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sum = sum
	s.count = count
}
