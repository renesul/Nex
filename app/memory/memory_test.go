package memory

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"nex/app/types"
)

func setupTestMemory(t *testing.T) (*Memory, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	mem, err := NewMemory(db)
	if err != nil {
		t.Fatal("NewMemory:", err)
	}
	return mem, db
}

func TestGetOrCreateSession(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	sid, isNew, oldSID := mem.GetOrCreateSession("5511999", 240)
	if !isNew {
		t.Error("first session should be new")
	}
	if oldSID != 0 {
		t.Errorf("first session oldSID = %d, want 0", oldSID)
	}
	if sid == 0 {
		t.Error("session ID should not be 0")
	}

	mem.SaveMessage("5511999", "user", "hello", sid)

	sid2, isNew2, _ := mem.GetOrCreateSession("5511999", 240)
	if isNew2 {
		t.Error("second call should return existing session")
	}
	if sid2 != sid {
		t.Errorf("session ID changed: %d != %d", sid2, sid)
	}
}

func TestSessionTimeout(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	chatID := "5511888"
	oldTime := time.Now().Unix() - 300*60
	db.Exec("INSERT INTO messages (chat_id, role, content, session_id, created_at) VALUES (?, 'user', 'old msg', ?, ?)",
		chatID, oldTime, oldTime)

	sid, isNew, oldSID := mem.GetOrCreateSession(chatID, 240)
	if !isNew {
		t.Error("should create new session after timeout")
	}
	if oldSID != oldTime {
		t.Errorf("oldSID = %d, want %d", oldSID, oldTime)
	}
	if sid == oldTime {
		t.Error("new session ID should differ from old")
	}
}

func TestSaveAndGetHistory(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	chatID := "5511777"
	sessionID := int64(1000)

	mem.SaveMessage(chatID, "user", "msg1", sessionID)
	mem.SaveMessage(chatID, "assistant", "reply1", sessionID)
	mem.SaveMessage(chatID, "user", "msg2", sessionID)

	history, err := mem.GetSessionHistory(chatID, sessionID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	if history[0].Content != "msg1" {
		t.Errorf("first message = %q, want msg1", history[0].Content)
	}
	if history[2].Content != "msg2" {
		t.Errorf("last message = %q, want msg2", history[2].Content)
	}

	history, _ = mem.GetSessionHistory(chatID, sessionID, 2)
	if len(history) != 2 {
		t.Errorf("limit=2 should return 2, got %d", len(history))
	}
}

func TestTrimToTokenBudget(t *testing.T) {
	msgs := []types.Message{
		{Content: "short"},
		{Content: "a medium length message right here now"},
		{Content: "final"},
	}

	result := TrimToTokenBudget(msgs, 1000)
	if len(result) != 3 {
		t.Errorf("no trim: got %d, want 3", len(result))
	}

	result = TrimToTokenBudget(msgs, 3)
	if len(result) > 2 {
		t.Errorf("should trim: got %d messages", len(result))
	}
	if result[len(result)-1].Content != "final" {
		t.Errorf("last = %q, want final", result[len(result)-1].Content)
	}
}

func TestEstimateTokens(t *testing.T) {
	msgs := []types.Message{
		{Content: "12345678"}, // 8/3 + 4 = 6
		{Content: "1234"},     // 4/3 + 4 = 5
	}
	got := EstimateTokens(msgs)
	want := 11 // 6 + 5
	if got != want {
		t.Errorf("EstimateTokens = %d, want %d", got, want)
	}

	if EstimateTokens(nil) != 0 {
		t.Error("empty should return 0")
	}
}

func TestGetLatestSummary(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	chatID := "5511666"

	if s := mem.GetLatestSummary(chatID); s != "" {
		t.Errorf("expected empty, got %q", s)
	}

	db.Exec("INSERT INTO summaries (chat_id, session_id, content, created_at) VALUES (?, 1, 'first summary', 1000)", chatID)
	db.Exec("INSERT INTO summaries (chat_id, session_id, content, created_at) VALUES (?, 2, 'second summary', 2000)", chatID)

	got := mem.GetLatestSummary(chatID)
	if got != "second summary" {
		t.Errorf("GetLatestSummary = %q, want 'second summary'", got)
	}
}

func TestGetContacts(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	mem.SaveMessage("5511111", "user", "hello from 111", 1)
	mem.SaveMessage("5511222", "user", "hello from 222", 2)
	mem.SaveMessage("5511111", "assistant", "reply to 111", 1)

	contacts, err := mem.GetContacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}

	if contacts[0].ChatID != "5511111" {
		t.Errorf("first contact = %q, want 5511111", contacts[0].ChatID)
	}
	if contacts[0].LastMessage != "reply to 111" {
		t.Errorf("last message = %q, want 'reply to 111'", contacts[0].LastMessage)
	}
}

func TestReadReceipts(t *testing.T) {
	mem, db := setupTestMemory(t)
	defer db.Close()

	chatID := "5511444"
	sessionID := int64(3000)

	mem.SaveMessage(chatID, "user", "oi", sessionID)
	mem.SaveMessage(chatID, "assistant", "ola!", sessionID)

	err := mem.SetLastAssistantWAMsgID(chatID, "WAMSG123")
	if err != nil {
		t.Fatal("SetLastAssistantWAMsgID:", err)
	}

	msgs, _ := mem.GetAllMessages(chatID)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[1].WAMsgID != "WAMSG123" {
		t.Errorf("wa_msg_id = %q, want WAMSG123", msgs[1].WAMsgID)
	}
	if msgs[1].ReadAt != 0 {
		t.Errorf("read_at should be 0 before marking, got %d", msgs[1].ReadAt)
	}

	err = mem.MarkReadByWAMsgID("WAMSG123")
	if err != nil {
		t.Fatal("MarkReadByWAMsgID:", err)
	}

	msgs, _ = mem.GetAllMessages(chatID)
	if msgs[1].ReadAt == 0 {
		t.Error("read_at should be >0 after marking")
	}

	err = mem.MarkReadByWAMsgID("WAMSG123")
	if err != nil {
		t.Fatal("second MarkReadByWAMsgID:", err)
	}

	err = mem.MarkReadByWAMsgID("UNKNOWN")
	if err != nil {
		t.Fatal("MarkReadByWAMsgID unknown:", err)
	}
}
