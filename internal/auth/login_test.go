package auth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/config"
)

// Issue #63: a phone asked to sign in again looked like a fault. The page says
// when it happens: a restart, the session's lifetime, or a change of address or
// browser -- the three things a session is bound to (session.go).
func TestLoginPageSaysWhenASignInEnds(t *testing.T) {
	page := LoginPage(false)
	for _, want := range []string{
		"lasts until GobboNet restarts",
		"12 hours at most by\n  default",
		"only for this device and browser",
		"changes network\n  address",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("login page does not say %q", want)
		}
	}
	// The plain-text warning is still there, and still first.
	plain := strings.Index(page, "in plain text")
	lasts := strings.Index(page, "A sign-in lasts")
	if plain < 0 || lasts < plain {
		t.Errorf("the plain-text note should come first (at %d, sign-in note at %d)", plain, lasts)
	}
}

// The figure the page quotes is the default the config ships with. If the
// default moves, the sentence has to move with it.
func TestLoginPageQuotesTheDefaultLifetime(t *testing.T) {
	want := fmt.Sprintf("%d hours at most", config.DefaultSessionTTLHours)
	if !strings.Contains(LoginPage(false), want) {
		t.Errorf("the login page does not say %q", want)
	}
}

// Issue #63: a phone usually meets GobboNet at this page, so "Add to Home
// Screen" from here needs the same name and icon as the chat page. Absolute
// paths, because this page is also served at whatever deep link a signed-out
// browser opened.
func TestLoginPageCarriesTheHomeScreenTags(t *testing.T) {
	page := LoginPage(false)
	if !strings.Contains(page, HomeScreenTags) {
		t.Fatal("the login page does not carry HomeScreenTags")
	}
	for _, want := range []string{
		`<link rel="manifest" href="/manifest.webmanifest">`,
		`<link rel="apple-touch-icon" href="/icons/gobbonet-180.png">`,
		`<meta name="apple-mobile-web-app-title" content="GobboNet">`,
	} {
		if !strings.Contains(HomeScreenTags, want) {
			t.Errorf("HomeScreenTags lacks %s", want)
		}
	}
	if strings.Contains(HomeScreenTags, "http") || strings.Contains(HomeScreenTags, `href="icons`) {
		t.Error("every path in HomeScreenTags is absolute and on this machine")
	}
	if strings.Index(page, HomeScreenTags) > strings.Index(page, "<style>") {
		t.Error("the tags belong in <head>, before the styles")
	}
}
