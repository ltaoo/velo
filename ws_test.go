package velo

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type testWSClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func TestWebSocketInvokesRegisteredRoute(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp})
	app.Get("/api/ws/ping", func(c *BoxContext) interface{} {
		var args struct {
			Message string `json:"message"`
		}
		if err := c.BindJSON(&args); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(H{
			"message": args.Message,
			"name":    c.Query("name"),
		})
	})

	server := httptest.NewServer(app.setupMux(nil, ""))
	defer server.Close()

	client := dialTestWS(t, server.URL)
	defer client.close()

	request := `{"id":"req-1","method":"/api/ws/ping?name=velo","headers":{"Content-Type":["application/json"]},"args":{"message":"pong"}}`
	client.writeText(t, []byte(request))

	_, opcode, payload, err := readWSFrame(client.reader)
	if err != nil {
		t.Fatalf("read websocket callback: %v", err)
	}
	if opcode != wsOpcodeText {
		t.Fatalf("opcode = %d, want text", opcode)
	}

	var frame struct {
		Type   string    `json:"type"`
		ID     string    `json:"id"`
		Result BoxResult `json:"result"`
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		t.Fatalf("unmarshal callback frame: %v; payload=%s", err, payload)
	}
	if frame.Type != veloWSCallbackType {
		t.Fatalf("frame type = %q, want %q", frame.Type, veloWSCallbackType)
	}
	if frame.ID != "req-1" {
		t.Fatalf("frame id = %q, want req-1", frame.ID)
	}
	if frame.Result.Code != 0 {
		t.Fatalf("result code = %d, want 0; msg=%q", frame.Result.Code, frame.Result.Msg)
	}
	data, ok := frame.Result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("result data type = %T, want object", frame.Result.Data)
	}
	if data["message"] != "pong" || data["name"] != "velo" {
		t.Fatalf("result data = %#v", data)
	}
}

func TestSendMessageBroadcastsToWebSocketClients(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp})

	server := httptest.NewServer(app.setupMux(nil, ""))
	defer server.Close()

	client := dialTestWS(t, server.URL)
	defer client.close()

	if ok := app.SendMessage(H{"type": "notice", "count": 2}); !ok {
		t.Fatal("SendMessage returned false")
	}

	_, opcode, payload, err := readWSFrame(client.reader)
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if opcode != wsOpcodeText {
		t.Fatalf("opcode = %d, want text", opcode)
	}

	var frame struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		t.Fatalf("unmarshal message frame: %v; payload=%s", err, payload)
	}
	if frame.Type != veloWSMessageType {
		t.Fatalf("frame type = %q, want %q", frame.Type, veloWSMessageType)
	}
	if frame.Payload["type"] != "notice" || frame.Payload["count"] != float64(2) {
		t.Fatalf("payload = %#v", frame.Payload)
	}
}

func dialTestWS(t *testing.T, serverURL string) *testWSClient {
	t.Helper()

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatalf("dial websocket server: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}

	var keyBytes [16]byte
	if _, err := rand.Read(keyBytes[:]); err != nil {
		t.Fatalf("generate websocket key: %v", err)
	}
	key := base64.StdEncoding.EncodeToString(keyBytes[:])
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n\r\n", VeloWebSocketPath, u.Host, key)
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write websocket handshake: %v", err)
	}

	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read websocket handshake status: %v", err)
	}
	if !strings.Contains(status, "101") {
		t.Fatalf("websocket status = %q, want 101", strings.TrimSpace(status))
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read websocket handshake header: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	return &testWSClient{conn: conn, reader: reader}
}

func (c *testWSClient) writeText(t *testing.T, payload []byte) {
	t.Helper()
	if err := writeMaskedWSFrame(c.conn, wsOpcodeText, payload); err != nil {
		t.Fatalf("write websocket text frame: %v", err)
	}
}

func (c *testWSClient) close() {
	_ = c.conn.Close()
}

func writeMaskedWSFrame(w io.Writer, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, 0x80|byte(length))
	case length <= 0xffff:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}

	mask := [4]byte{1, 2, 3, 4}
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}

	if _, err := w.Write(header); err != nil {
		return err
	}
	if _, err := w.Write(mask[:]); err != nil {
		return err
	}
	_, err := w.Write(masked)
	return err
}
