package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeNextMatchInput(t *testing.T) {
	got, err := normalizeNextMatchInput(nextMatchBody{
		Date: " 2026-09-05 ", Time: "19:30", Location: "  Club   Central · Court 2 ",
	})
	if err != nil {
		t.Fatalf("normalizeNextMatchInput returned error: %v", err)
	}
	if got.Date != "2026-09-05" || got.Time != "19:30" || got.Location != "Club Central · Court 2" {
		t.Fatalf("unexpected normalized input: %#v", got)
	}
}

func TestNormalizeNextMatchInputRejectsInvalidFields(t *testing.T) {
	tests := []nextMatchBody{
		{Date: "09/05/2026", Time: "19:30", Location: "Club"},
		{Date: "2026-09-05", Time: "7:30 PM", Location: "Club"},
		{Date: "2026-09-05", Time: "19:30", Location: "   "},
		{Date: "2026-09-05", Time: "19:30", Location: strings.Repeat("x", 161)},
	}
	for _, input := range tests {
		if _, err := normalizeNextMatchInput(input); err == nil {
			t.Fatalf("expected invalid input to fail: %#v", input)
		}
	}
}

func TestNextMatchFrontendContract(t *testing.T) {
	readWeb := func(name string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", "web", name))
		if err != nil {
			t.Fatalf("read web/%s: %v", name, err)
		}
		return string(body)
	}

	html := readWeb("index.html")
	for _, required := range []string{
		`id="next-match-card" class="next-match-card" hidden`,
		`id="manage-next-match" class="menu__item admin-only"`,
		`id="next-match-input-date"`,
		`id="next-match-input-time"`,
		`id="next-match-input-location"`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("web/index.html missing %q", required)
		}
	}

	api := readWeb("api.js")
	for _, method := range []string{"getNextMatch", "saveNextMatch", "clearNextMatch"} {
		if !strings.Contains(api, method) {
			t.Fatalf("web/api.js missing %s", method)
		}
	}

	app := readWeb("app.js")
	if !strings.Contains(app, "els.nextMatchCard.hidden = !next") {
		t.Fatal("next match card must remain hidden when no schedule exists")
	}
	if !strings.Contains(app, "openNextMatchEditor(els.manageNextMatch)") {
		t.Fatal("next match editor must be opened from the settings menu")
	}
}

func TestFrontendDoesNotRequireInlineStyleCSPExceptions(t *testing.T) {
	for _, name := range []string{"index.html", "app.js", "api.js", "state.js", "ui-enhancements.js"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "web", name))
		if err != nil {
			t.Fatalf("read web/%s: %v", name, err)
		}
		text := string(body)
		if strings.Contains(text, `style=`) || strings.Contains(text, `.style.`) {
			t.Fatalf("web/%s uses an inline style that the Content Security Policy blocks", name)
		}
	}
}
