package rate

import (
    "context"
    "sync"
    "time"
)

// 简易内存限速器：支持 RPS、RPM、Burst、并发限制。

type Limiter struct {
    mu          sync.Mutex
    rps         int
    rpm         int
    burst       int
    concurrency int

    tokens      int
    lastRefill  time.Time

    // RPM window
    minuteCount int
    minuteStart time.Time

    sem         chan struct{}
}

func NewLimiter(rps, rpm, burst, concurrency int) *Limiter {
    if burst <= 0 { burst = rps }
    if concurrency <= 0 { concurrency = 8 }
    l := &Limiter{
        rps:         rps,
        rpm:         rpm,
        burst:       burst,
        concurrency: concurrency,
        tokens:      burst,
        lastRefill:  time.Now(),
        minuteStart: time.Now(),
        sem:         make(chan struct{}, concurrency),
    }
    return l
}

func (l *Limiter) tryTakeTokenLocked(now time.Time) bool {
    // refill per second
    if l.rps > 0 {
        elapsed := now.Sub(l.lastRefill)
        if elapsed >= time.Second {
            n := int(elapsed / time.Second) * l.rps
            l.tokens += n
            if l.tokens > l.burst { l.tokens = l.burst }
            l.lastRefill = now
        }
        if l.tokens <= 0 { return false }
        l.tokens--
    }
    // RPM check
    if l.rpm > 0 {
        if now.Sub(l.minuteStart) >= time.Minute {
            l.minuteStart = now
            l.minuteCount = 0
        }
        if l.minuteCount >= l.rpm { return false }
        l.minuteCount++
    }
    return true
}

func (l *Limiter) Acquire(ctx context.Context) error {
    // concurrency
    select {
    case l.sem <- struct{}{}:
        // proceed
    case <-ctx.Done():
        return ctx.Err()
    }

    // tokens
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()
    for {
        now := time.Now()
        l.mu.Lock()
        ok := l.tryTakeTokenLocked(now)
        l.mu.Unlock()
        if ok { return nil }
        select {
        case <-ctx.Done():
            l.Release()
            return ctx.Err()
        case <-ticker.C:
        }
    }
}

func (l *Limiter) Release() {
    select {
    case <-l.sem:
    default:
    }
}

type LimiterManager struct {
    mu       sync.RWMutex
    limiters map[string]*Limiter
}

func NewLimiterManager() *LimiterManager {
    return &LimiterManager{limiters: make(map[string]*Limiter)}
}

func (m *LimiterManager) Configure(id string, rps, rpm, burst, concurrency int) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.limiters[id] = NewLimiter(rps, rpm, burst, concurrency)
}

func (m *LimiterManager) Get(id string) *Limiter {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.limiters[id]
}

func (m *LimiterManager) Remove(id string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    delete(m.limiters, id)
}

