package debounce

import (
	"sync"
	"testing"
	"time"
)

func TestSingleMessage(t *testing.T) {
	var mu sync.Mutex
	var result string
	done := make(chan struct{})

	d := NewDebouncer(50*time.Millisecond, 500*time.Millisecond, func(chatID, text, pushName string) {
		mu.Lock()
		result = chatID + ":" + text
		mu.Unlock()
		close(done)
	})

	d.Add("5511999", "hello", "João")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not fire within 1s")
	}

	mu.Lock()
	if result != "5511999:hello" {
		t.Errorf("result = %q, want 5511999:hello", result)
	}
	mu.Unlock()
}

func TestMultipleMessages(t *testing.T) {
	var mu sync.Mutex
	var result string
	done := make(chan struct{})

	d := NewDebouncer(100*time.Millisecond, 2*time.Second, func(chatID, text, pushName string) {
		mu.Lock()
		result = text
		mu.Unlock()
		close(done)
	})

	d.Add("phone1", "msg1", "Test")
	time.Sleep(30 * time.Millisecond)
	d.Add("phone1", "msg2", "Test")
	time.Sleep(30 * time.Millisecond)
	d.Add("phone1", "msg3", "Test")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not fire within 1s")
	}

	mu.Lock()
	if result != "msg1\nmsg2\nmsg3" {
		t.Errorf("result = %q, want msg1\\nmsg2\\nmsg3", result)
	}
	mu.Unlock()
}

func TestMaxWait(t *testing.T) {
	var mu sync.Mutex
	var result string
	done := make(chan struct{}, 1)

	d := NewDebouncer(200*time.Millisecond, 150*time.Millisecond, func(chatID, text, pushName string) {
		mu.Lock()
		result = text
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	})

	d.Add("phone1", "msg1", "Test")
	time.Sleep(160 * time.Millisecond)
	d.Add("phone1", "msg2", "Test")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not fire within 1s")
	}

	mu.Lock()
	got := result
	mu.Unlock()

	if got != "msg1\nmsg2" {
		t.Errorf("result = %q, want msg1\\nmsg2", got)
	}
}

func TestUpdateTimings(t *testing.T) {
	d := NewDebouncer(100*time.Millisecond, 500*time.Millisecond, func(chatID, text, pushName string) {})

	d.UpdateTimings(200*time.Millisecond, 1000*time.Millisecond)

	d.mu.Lock()
	if d.wait != 200*time.Millisecond {
		t.Errorf("wait = %v, want 200ms", d.wait)
	}
	if d.maxWait != 1000*time.Millisecond {
		t.Errorf("maxWait = %v, want 1000ms", d.maxWait)
	}
	d.mu.Unlock()
}

func TestStop(t *testing.T) {
	var mu sync.Mutex
	called := false

	d := NewDebouncer(100*time.Millisecond, 500*time.Millisecond, func(chatID, text, pushName string) {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	d.Add("phone-stop", "should not fire", "Test")
	d.Stop()

	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	if called {
		t.Error("handler should NOT have been called after Stop")
	}
	mu.Unlock()
}

func TestStop_Empty(t *testing.T) {
	d := NewDebouncer(100*time.Millisecond, 500*time.Millisecond, func(chatID, text, pushName string) {})
	d.Stop() // must not panic
}
