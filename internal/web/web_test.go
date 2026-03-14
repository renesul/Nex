package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- cleanSQL tests ---

func TestCleanSQL_PlainSQL(t *testing.T) {
	input := "SELECT * FROM users"
	got := cleanSQL(input)
	if got != input {
		t.Errorf("cleanSQL(%q) = %q, want %q", input, got, input)
	}
}

func TestCleanSQL_WithMarkdownFences(t *testing.T) {
	input := "```sql\nSELECT * FROM users\n```"
	want := "SELECT * FROM users"
	got := cleanSQL(input)
	if got != want {
		t.Errorf("cleanSQL(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanSQL_WithPlainFences(t *testing.T) {
	input := "```\nSELECT 1\n```"
	want := "SELECT 1"
	got := cleanSQL(input)
	if got != want {
		t.Errorf("cleanSQL(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanSQL_WhitespaceOnly(t *testing.T) {
	got := cleanSQL("   \n  ")
	if got != "" {
		t.Errorf("cleanSQL(whitespace) = %q, want empty", got)
	}
}

func TestCleanSQL_MultilineSQLInFences(t *testing.T) {
	input := "```sql\nSELECT id,\n  name\nFROM users\nWHERE active = 1\n```"
	want := "SELECT id,\n  name\nFROM users\nWHERE active = 1"
	got := cleanSQL(input)
	if got != want {
		t.Errorf("cleanSQL = %q, want %q", got, want)
	}
}

// --- formatDataSummary tests ---

func TestFormatDataSummary_Basic(t *testing.T) {
	cols := []string{"id", "name"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}
	got := formatDataSummary(cols, rows)
	if !strings.HasPrefix(got, "id | name\n") {
		t.Errorf("expected header line, got %q", got)
	}
	if !strings.Contains(got, "1 | Alice") {
		t.Error("missing row 1")
	}
	if !strings.Contains(got, "2 | Bob") {
		t.Error("missing row 2")
	}
}

func TestFormatDataSummary_EmptyRows(t *testing.T) {
	cols := []string{"col"}
	got := formatDataSummary(cols, nil)
	want := "col\n"
	if got != want {
		t.Errorf("formatDataSummary(empty) = %q, want %q", got, want)
	}
}

func TestFormatDataSummary_TruncatesAt50(t *testing.T) {
	cols := []string{"n"}
	rows := make([][]string, 60)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("%d", i)}
	}
	got := formatDataSummary(cols, rows)
	if !strings.Contains(got, "... (60 linhas no total)") {
		t.Error("expected truncation message for 60 rows")
	}
	// Should have header + 50 data rows + truncation line = 52 lines
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 52 {
		t.Errorf("expected 52 lines, got %d", len(lines))
	}
}

func TestFormatDataSummary_Exactly50(t *testing.T) {
	cols := []string{"n"}
	rows := make([][]string, 50)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("%d", i)}
	}
	got := formatDataSummary(cols, rows)
	if strings.Contains(got, "...") {
		t.Error("should not truncate at exactly 50 rows")
	}
}

// --- parseInterpretation tests ---

func TestParseInterpretation_ValidJSON(t *testing.T) {
	raw := `{"insight":"Sales are up","chart_type":"bar","chart_config":{"title":"Sales"}}`
	var resp reportResponse
	parseInterpretation(raw, &resp)

	if resp.Insight != "Sales are up" {
		t.Errorf("Insight = %q, want %q", resp.Insight, "Sales are up")
	}
	if resp.ChartType != "bar" {
		t.Errorf("ChartType = %q, want %q", resp.ChartType, "bar")
	}
	if resp.ChartConfig["title"] != "Sales" {
		t.Errorf("ChartConfig[title] = %v", resp.ChartConfig["title"])
	}
}

func TestParseInterpretation_JSONInMarkdownFence(t *testing.T) {
	raw := "```json\n{\"insight\":\"ok\",\"chart_type\":\"pie\",\"chart_config\":{}}\n```"
	var resp reportResponse
	parseInterpretation(raw, &resp)

	if resp.Insight != "ok" {
		t.Errorf("Insight = %q, want %q", resp.Insight, "ok")
	}
	if resp.ChartType != "pie" {
		t.Errorf("ChartType = %q, want %q", resp.ChartType, "pie")
	}
}

func TestParseInterpretation_InvalidJSON_FallsBackToRaw(t *testing.T) {
	raw := "This is just plain text"
	var resp reportResponse
	parseInterpretation(raw, &resp)

	if resp.Insight != raw {
		t.Errorf("Insight = %q, want raw text %q", resp.Insight, raw)
	}
	if resp.ChartType != "none" {
		t.Errorf("ChartType = %q, want %q", resp.ChartType, "none")
	}
}

func TestParseInterpretation_InvalidChartType_DefaultsToNone(t *testing.T) {
	raw := `{"insight":"test","chart_type":"scatter","chart_config":{}}`
	var resp reportResponse
	parseInterpretation(raw, &resp)

	if resp.ChartType != "none" {
		t.Errorf("ChartType = %q, want %q for invalid type", resp.ChartType, "none")
	}
}

func TestParseInterpretation_AllValidChartTypes(t *testing.T) {
	for _, ct := range []string{"bar", "line", "pie", "doughnut", "none"} {
		raw := fmt.Sprintf(`{"insight":"x","chart_type":"%s","chart_config":{}}`, ct)
		var resp reportResponse
		parseInterpretation(raw, &resp)
		if resp.ChartType != ct {
			t.Errorf("ChartType = %q, want %q", resp.ChartType, ct)
		}
	}
}

func TestParseInterpretation_EmptyString(t *testing.T) {
	var resp reportResponse
	parseInterpretation("", &resp)

	if resp.Insight != "" {
		t.Errorf("Insight = %q, want empty", resp.Insight)
	}
	if resp.ChartType != "none" {
		t.Errorf("ChartType = %q, want %q", resp.ChartType, "none")
	}
}

func TestStaticFileServing(t *testing.T) {
	// Create a temp static dir with a CSS file
	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	os.MkdirAll(staticDir, 0755)
	os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{color:red}"), 0644)

	handler := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
	mux := http.NewServeMux()
	mux.Handle("/static/", handler)

	// Existing file returns 200
	req := httptest.NewRequest("GET", "/static/style.css", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("GET /static/style.css: expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("expected Content-Type text/css, got %q", ct)
	}

	// Non-existent file returns 404
	req2 := httptest.NewRequest("GET", "/static/nope.js", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 404 {
		t.Fatalf("GET /static/nope.js: expected 404, got %d", rr2.Code)
	}
}
