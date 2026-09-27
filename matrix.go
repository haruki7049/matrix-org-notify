package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// httpClient is used for all requests to the Matrix homeserver. A timeout
// is set so a stuck connection can't hang the CLI forever.
var httpClient = &http.Client{Timeout: 15 * time.Second}

type sendMessageRequest struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
}

// sendMessage delivers message to cfg.RoomID via the Matrix Client-Server
// API: PUT /_matrix/client/v3/rooms/{roomId}/send/m.room.message/{txnId}.
func sendMessage(client *http.Client, cfg config, message string) error {
	reqURL := fmt.Sprintf(
		"%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		strings.TrimRight(cfg.Homeserver, "/"),
		url.PathEscape(cfg.RoomID),
		url.PathEscape(transactionID()),
	)

	payload, err := json.Marshal(sendMessageRequest{MsgType: "m.text", Body: message})
	if err != nil {
		return fmt.Errorf("encoding message payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to matrix homeserver: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("matrix homeserver returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

// txnCounter is combined with a timestamp in transactionID to guarantee
// uniqueness even when called twice within the same clock tick: some
// platforms (observed on macOS CI runners) have a coarser time.Now()
// resolution than a nanosecond.
var txnCounter atomic.Uint64

// transactionID returns a value that is unique per call, suitable for use
// as a Matrix Client-Server API transaction ID.
func transactionID() string {
	n := txnCounter.Add(1)
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(n, 10)
}
