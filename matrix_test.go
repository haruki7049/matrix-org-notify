package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMessage(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody sendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"event_id":"$abc123"}`))
	}))
	defer server.Close()

	cfg := config{
		Homeserver:  server.URL,
		RoomID:      "!room:example.org",
		AccessToken: "test-token",
	}

	if err := sendMessage(server.Client(), cfg, "hello matrix"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.HasPrefix(gotPath, "/_matrix/client/v3/rooms/!room:example.org/send/m.room.message/") {
		t.Errorf("path = %q, unexpected prefix", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotBody.MsgType != "m.text" || gotBody.Body != "hello matrix" {
		t.Errorf("body = %+v, unexpected content", gotBody)
	}
}

func TestSendMessageErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errcode":"M_FORBIDDEN"}`))
	}))
	defer server.Close()

	cfg := config{Homeserver: server.URL, RoomID: "!room:example.org", AccessToken: "bad-token"}

	err := sendMessage(server.Client(), cfg, "hello")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %q, want it to mention status 403", err.Error())
	}
}

func TestTransactionIDIsUnique(t *testing.T) {
	a := transactionID()
	b := transactionID()
	if a == b {
		t.Fatalf("transactionID returned the same value twice: %q", a)
	}
}
