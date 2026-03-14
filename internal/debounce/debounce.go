package debounce

import (
	"strings"
	"sync"
	"time"
)

// Debouncer groups rapid sequential messages per chat ID.
type Debouncer struct {
	mu      sync.Mutex
	pending map[string]*pendingMsg
	handler func(chatID, text, pushName string)
	wait    time.Duration
	maxWait time.Duration
}

type pendingMsg struct {
	texts    []string
	pushName string
	timer    *time.Timer
	started  time.Time
}

// NewDebouncer creates a debouncer with the given wait times and handler.
func NewDebouncer(wait, maxWait time.Duration, handler func(chatID, text, pushName string)) *Debouncer {
	return &Debouncer{
		pending: make(map[string]*pendingMsg),
		handler: handler,
		wait:    wait,
		maxWait: maxWait,
	}
}

// UpdateTimings changes the debounce wait times.
func (d *Debouncer) UpdateTimings(wait, maxWait time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wait = wait
	d.maxWait = maxWait
}

// Add adds a message text for a chat ID. When the debounce timer fires,
// all accumulated texts are joined with newlines and passed to the handler.
func (d *Debouncer) Add(chatID, text, pushName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	p, exists := d.pending[chatID]
	if !exists {
		p = &pendingMsg{
			texts:    []string{text},
			pushName: pushName,
			started:  time.Now(),
		}
		p.timer = time.AfterFunc(d.wait, func() {
			d.fire(chatID)
		})
		d.pending[chatID] = p
		return
	}

	p.texts = append(p.texts, text)
	if pushName != "" {
		p.pushName = pushName
	}

	// Check if max wait exceeded
	if time.Since(p.started) >= d.maxWait {
		p.timer.Stop()
		go d.fire(chatID)
		return
	}

	// Reset timer
	p.timer.Stop()
	p.timer = time.AfterFunc(d.wait, func() {
		d.fire(chatID)
	})
}

func (d *Debouncer) fire(chatID string) {
	d.mu.Lock()
	p, exists := d.pending[chatID]
	if !exists {
		d.mu.Unlock()
		return
	}
	combined := strings.Join(p.texts, "\n")
	pushName := p.pushName
	delete(d.pending, chatID)
	d.mu.Unlock()

	d.handler(chatID, combined, pushName)
}
