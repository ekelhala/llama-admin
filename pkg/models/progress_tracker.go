package models

import (
	"sync/atomic"
)

type ProgressTracker struct {
	downloaded atomic.Int64
	total      atomic.Int64
}

func (p *ProgressTracker) SetTotal(total int64) {
	p.total.Store(total)
}

func (p *ProgressTracker) Add(n int64) {
	p.downloaded.Add(n)
}

func (p *ProgressTracker) Snapshot() (downloaded, total int64, percent float64) {
	d := p.downloaded.Load()
	t := p.total.Load()
	if t > 0 {
		percent = float64(d) / float64(t) * 100
	}
	return d, t, percent
}
