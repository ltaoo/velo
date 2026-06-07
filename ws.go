package velo

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const (
	// VeloWebSocketPath is the internal WebSocket endpoint used by the JS runtime.
	VeloWebSocketPath = "/__velo/ws"
	// VeloRuntimePath is the internal JS runtime endpoint used by HTTP-mode pages.
	VeloRuntimePath = "/__velo/runtime.js"

	veloWSCallbackType = "__velo_callback"
	veloWSMessageType  = "__velo_message"

	wsOpcodeContinuation = 0x0
	wsOpcodeText         = 0x1
	wsOpcodeBinary       = 0x2
	wsOpcodeClose        = 0x8
	wsOpcodePing         = 0x9
	wsOpcodePong         = 0xa

	maxWSMessageSize = 16 << 20
)

type veloWSHub struct {
	mu      sync.RWMutex
	clients map[*veloWSConn]struct{}
}

type veloWSConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writeMu sync.Mutex
	once    sync.Once
}

func newVeloWSHub() *veloWSHub {
	return &veloWSHub{
		clients: make(map[*veloWSConn]struct{}),
	}
}

func (h *veloWSHub) ServeHTTP(w http.ResponseWriter, r *http.Request, handleMessage func(string) (string, string)) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !headerContains(r.Header, "Connection", "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijacking unavailable", http.StatusInternalServerError)
		return
	}

	netConn, rw, err := hijacker.Hijack()
	if err != nil {
		return
	}

	accept := computeWSAccept(key)
	_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	_, _ = rw.WriteString("Upgrade: websocket\r\n")
	_, _ = rw.WriteString("Connection: Upgrade\r\n")
	_, _ = rw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
	if err := rw.Flush(); err != nil {
		_ = netConn.Close()
		return
	}

	client := &veloWSConn{conn: netConn, reader: rw.Reader}
	h.add(client)
	defer func() {
		h.remove(client)
		client.close()
	}()

	if handleMessage == nil {
		return
	}

	var fragmentedOpcode byte
	var fragmentedPayload []byte

	for {
		fin, opcode, payload, err := readWSFrame(client.reader)
		if err != nil {
			return
		}

		switch opcode {
		case wsOpcodeText, wsOpcodeBinary:
			if !fin {
				fragmentedOpcode = opcode
				fragmentedPayload = append(fragmentedPayload[:0], payload...)
				continue
			}
			go h.handleClientMessage(client, string(payload), handleMessage)
		case wsOpcodeContinuation:
			if fragmentedOpcode == 0 {
				return
			}
			fragmentedPayload = append(fragmentedPayload, payload...)
			if len(fragmentedPayload) > maxWSMessageSize {
				return
			}
			if fin {
				msg := string(fragmentedPayload)
				fragmentedOpcode = 0
				fragmentedPayload = nil
				go h.handleClientMessage(client, msg, handleMessage)
			}
		case wsOpcodePing:
			_ = client.writeFrame(wsOpcodePong, payload)
		case wsOpcodePong:
			continue
		case wsOpcodeClose:
			_ = client.writeFrame(wsOpcodeClose, nil)
			return
		default:
			return
		}
	}
}

func (h *veloWSHub) BroadcastMessage(message interface{}) bool {
	if h == nil {
		return false
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return false
	}
	frame, err := makeWSMessageFrame(payload)
	if err != nil {
		return false
	}
	return h.broadcastText(frame)
}

func (h *veloWSHub) handleClientMessage(client *veloWSConn, message string, handleMessage func(string) (string, string)) {
	id, result := handleMessage(message)
	if id == "" {
		return
	}
	frame, err := makeWSCallbackFrame(id, result)
	if err != nil {
		return
	}
	if err := client.writeText(frame); err != nil {
		client.close()
		h.remove(client)
	}
}

func (h *veloWSHub) broadcastText(payload []byte) bool {
	h.mu.RLock()
	clients := make([]*veloWSConn, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	delivered := false
	for _, client := range clients {
		if err := client.writeText(payload); err != nil {
			client.close()
			h.remove(client)
			continue
		}
		delivered = true
	}
	return delivered
}

func (h *veloWSHub) add(client *veloWSConn) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *veloWSHub) remove(client *veloWSConn) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
}

func (c *veloWSConn) writeText(payload []byte) error {
	return c.writeFrame(wsOpcodeText, payload)
}

func (c *veloWSConn) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return writeWSFrame(c.conn, opcode, payload)
}

func (c *veloWSConn) close() {
	c.once.Do(func() {
		_ = c.conn.Close()
	})
}

func makeWSCallbackFrame(id, result string) ([]byte, error) {
	if json.Valid([]byte(result)) {
		return json.Marshal(struct {
			Type   string          `json:"type"`
			ID     string          `json:"id"`
			Result json.RawMessage `json:"result"`
		}{
			Type:   veloWSCallbackType,
			ID:     id,
			Result: json.RawMessage(result),
		})
	}

	return json.Marshal(struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		Result string `json:"result"`
	}{
		Type:   veloWSCallbackType,
		ID:     id,
		Result: result,
	})
}

func makeWSMessageFrame(payload []byte) ([]byte, error) {
	return json.Marshal(struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}{
		Type:    veloWSMessageType,
		Payload: json.RawMessage(payload),
	})
}

func computeWSAccept(key string) string {
	const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	sum := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func headerContains(header http.Header, name, value string) bool {
	for _, item := range header.Values(name) {
		for _, part := range strings.Split(item, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return true
			}
		}
	}
	return false
}

func readWSFrame(r *bufio.Reader) (bool, byte, []byte, error) {
	first, err := r.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}
	second, err := r.ReadByte()
	if err != nil {
		return false, 0, nil, err
	}

	fin := first&0x80 != 0
	opcode := first & 0x0f
	masked := second&0x80 != 0
	length := uint64(second & 0x7f)

	switch length {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf[:]))
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(buf[:])
	}

	if length > maxWSMessageSize {
		return false, 0, nil, fmt.Errorf("websocket message too large: %d", length)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(r, maskKey[:]); err != nil {
			return false, 0, nil, err
		}
	}

	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return fin, opcode, payload, nil
}

func writeWSFrame(w io.Writer, opcode byte, payload []byte) error {
	if opcode > 0x0f {
		return errors.New("invalid websocket opcode")
	}

	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, byte(length))
	case length <= 0xffff:
		header = append(header, 126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}

	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}
