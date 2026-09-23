package notifier

import (
	"context"
	"testing"
)

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello & World", "Hello &amp; World"},
		{"<alert>", "&lt;alert&gt;"},
		{`"Quote" and 'Apos'`, "&quot;Quote&quot; and &apos;Apos&apos;"},
		{"Plain text", "Plain text"},
	}

	for _, tt := range tests {
		got := escapeXML(tt.input)
		if got != tt.want {
			t.Fatalf("escapeXML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeQuotes(t *testing.T) {
	input := `He said "hello" \ there`
	want := `He said \"hello\" \\ there`
	got := escapeQuotes(input)
	if got != want {
		t.Fatalf("escapeQuotes(%q) = %q, want %q", input, got, want)
	}
}

func TestMockNotifier(t *testing.T) {
	m := &MockNotifier{}
	ctx := context.Background()

	if err := m.Notify(ctx, "Posture Alert", "Shoulders dropped"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := m.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Title != "Posture Alert" || calls[0].Message != "Shoulders dropped" {
		t.Fatalf("unexpected call recorded: %+v", calls[0])
	}
}
