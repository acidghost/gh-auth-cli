package auth

import (
	"testing"
	"time"

	"github.com/acidghost/gh-auth-cli/internal/store"
)

func TestValidateCallbackURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "IPv4 loopback", value: "http://127.0.0.1:17836/callback"},
		{name: "localhost", value: "http://localhost:17836/callback"},
		{name: "IPv6 loopback", value: "http://[::1]:17836/callback"},
		{name: "HTTPS rejected", value: "https://127.0.0.1:17836/callback", wantErr: true},
		{name: "non-loopback rejected", value: "http://example.com:17836/callback", wantErr: true},
		{name: "missing port", value: "http://127.0.0.1/callback", wantErr: true},
		{name: "missing path", value: "http://127.0.0.1:17836/", wantErr: true},
		{name: "query rejected", value: "http://127.0.0.1:17836/callback?x=1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCallbackURL(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateCallbackURL(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
		})
	}
}

func TestUsable(t *testing.T) {
	t.Parallel()
	if !Usable(&store.StoredToken{AccessToken: "token"}, time.Hour) {
		t.Fatal("non-expiring token should be usable")
	}
	if Usable(&store.StoredToken{AccessToken: "token", AccessExpiresAt: time.Now().Add(5 * time.Minute)}, 10*time.Minute) {
		t.Fatal("token expiring before minimum validity should not be usable")
	}
	if !Usable(&store.StoredToken{AccessToken: "token", AccessExpiresAt: time.Now().Add(time.Hour)}, 10*time.Minute) {
		t.Fatal("token with sufficient validity should be usable")
	}
}

func TestDurationExpiry(t *testing.T) {
	t.Parallel()
	received := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := received.Add(90 * time.Second)
	for _, value := range []any{float64(90), "90", int64(90), 90} {
		if got := durationExpiry(value, received); !got.Equal(want) {
			t.Fatalf("durationExpiry(%v) = %s, want %s", value, got, want)
		}
	}
	if got := durationExpiry("invalid", received); !got.IsZero() {
		t.Fatalf("durationExpiry(invalid) = %s, want zero", got)
	}
}
