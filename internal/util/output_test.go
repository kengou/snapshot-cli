package util

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

// writeJSON

func TestWriteJSON_String(t *testing.T) {
	out := captureStdout(t, func() {
		if err := writeJSON("hello"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	out = strings.TrimSpace(out)
	if out != `"hello"` {
		t.Errorf("got %q, want %q", out, `"hello"`)
	}
}

func TestWriteJSON_Map(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2}
	out := captureStdout(t, func() {
		if err := writeJSON(input); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	var got map[string]int
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("output not valid JSON: %v\noutput: %s", err, out)
	}
	if got["a"] != 1 || got["b"] != 2 {
		t.Errorf("got %v, want {a:1 b:2}", got)
	}
}

func TestWriteJSON_Slice(t *testing.T) {
	input := []string{"x", "y", "z"}
	out := captureStdout(t, func() {
		if err := writeJSON(input); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	var got []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("output not valid JSON: %v\noutput: %s", err, out)
	}
	if len(got) != 3 || got[0] != "x" || got[2] != "z" {
		t.Errorf("got %v, want [x y z]", got)
	}
}

func TestWriteJSON_Nil(t *testing.T) {
	out := captureStdout(t, func() {
		if err := writeJSON(nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(out) != "null" {
		t.Errorf("got %q, want %q", strings.TrimSpace(out), "null")
	}
}

func TestWriteJSON_Struct(t *testing.T) {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	out := captureStdout(t, func() {
		if err := writeJSON(point{X: 3, Y: 7}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	var got point
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if got.X != 3 || got.Y != 7 {
		t.Errorf("got %+v, want {X:3 Y:7}", got)
	}
}

func TestWriteJSON_OutputEndsWithNewline(t *testing.T) {
	out := captureStdout(t, func() {
		if err := writeJSON("test"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("expected output to end with newline, got %q", out)
	}
}

func TestWriteJSON_UnmarshalableType_ReturnsError(t *testing.T) {
	// channels cannot be marshalled to JSON
	ch := make(chan int)
	err := writeJSON(ch)
	if err == nil {
		t.Error("expected error for unmarshalable type, got nil")
	}
	if !strings.Contains(err.Error(), "could not marshal JSON") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// isSlice

func TestIsSlice_Slice_ReturnsTrue(t *testing.T) {
	if !isSlice([]string{"a", "b"}) {
		t.Error("expected true for []string")
	}
	if !isSlice([]int{1, 2, 3}) {
		t.Error("expected true for []int")
	}
	if !isSlice([]any{}) {
		t.Error("expected true for []any")
	}
}

func TestIsSlice_NonSlice_ReturnsFalse(t *testing.T) {
	type myStruct struct{ A int }
	cases := []any{
		nil,
		"string",
		42,
		map[string]int{"a": 1},
		myStruct{A: 1},
		true,
	}
	for _, c := range cases {
		if isSlice(c) {
			t.Errorf("expected false for %T (%v)", c, c)
		}
	}
}

// writeAsTable

func TestWriteAsTable_Struct(t *testing.T) {
	type row struct {
		Name  string
		Value int
	}
	out := captureStdout(t, func() {
		if err := writeAsTable(row{Name: "foo", Value: 42}, []string{"Name", "Value"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "foo") {
		t.Errorf("expected 'foo' in table output, got: %s", out)
	}
}

func TestWriteAsTable_SliceOfStructs(t *testing.T) {
	type row struct {
		ID   string
		Size int
	}
	rows := []row{
		{ID: "abc-123", Size: 10},
		{ID: "def-456", Size: 20},
	}
	out := captureStdout(t, func() {
		if err := writeAsTable(rows, []string{"ID", "Size"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "abc-123") {
		t.Errorf("expected 'abc-123' in output, got: %s", out)
	}
	if !strings.Contains(out, "def-456") {
		t.Errorf("expected 'def-456' in output, got: %s", out)
	}
}

// FuzzwriteJSON verifies that writeJSON never panics for arbitrary string input.
func FuzzWriteJSON(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add(`{"key":"value"}`)
	f.Add("null")
	f.Add("\x00\xff\n\t")

	f.Fuzz(func(t *testing.T, s string) {
		// writeJSON must never panic regardless of input.
		// It may return an error for types it can't marshal, but strings always marshal.
		captureStdout(t, func() {
			if err := writeJSON(s); err != nil {
				t.Errorf("writeJSON(%q) returned unexpected error: %v", s, err)
			}
		})
	})
}

// Constants

func TestOutputConstants(t *testing.T) {
	if outputJSON != "json" {
		t.Errorf("outputJSON = %q, want %q", outputJSON, "json")
	}
	if outputTable != "table" {
		t.Errorf("outputTable = %q, want %q", outputTable, "table")
	}
}

// Render

func TestRender_JSON_DelegatesToWriteJSON(t *testing.T) {
	type row struct {
		ID string `json:"id"`
	}
	out := captureStdout(t, func() {
		if err := Render(outputJSON, row{ID: "abc"}, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, `"id":"abc"`) {
		t.Errorf("expected id in JSON output, got: %s", out)
	}
}

func TestRender_Table_DelegatesToWriteAsTable(t *testing.T) {
	type row struct {
		ID string
	}
	out := captureStdout(t, func() {
		if err := Render(outputTable, row{ID: "xyz"}, []string{"ID"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "xyz") {
		t.Errorf("expected ID in table output, got: %s", out)
	}
}

func TestRender_Unsupported_ReturnsError(t *testing.T) {
	err := Render("yaml", struct{}{}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error should mention unsupported format, got: %v", err)
	}
	if !strings.Contains(err.Error(), `"yaml"`) {
		t.Errorf("error should include the offending format string, got: %v", err)
	}
}

func TestRender_Empty_ReturnsError(t *testing.T) {
	if err := Render("", struct{}{}, nil); err == nil {
		t.Error("expected error for empty output format")
	}
}
