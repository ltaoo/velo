package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ltaoo/velo"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/rs/zerolog"
)

//go:embed frontend
var frontendFolder embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

//go:embed assets/WebView2Loader.dll
var webview2LoaderDLL []byte

var Version = "1.0.0"

const (
	defaultPortA = 49310
	defaultPortB = 49311
)

var (
	veloModeFlag   = flag.String("velo-mode", firstText(os.Getenv("VELO_IM_MODE"), "bridge"), "velo mode: bridge, bridge-http, http")
	wsSelfTestFlag = flag.Bool("ws-self-test", false, "run ModeHttp WebSocket transport self-test and exit")
)

type runtimeConfig struct {
	InstanceID  string `json:"instance_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	ListenPort  int    `json:"listen_port"`
	PeerPort    int    `json:"peer_port"`
	ListenURL   string `json:"listen_url"`
	PeerURL     string `json:"peer_url"`
	StartedAt   string `json:"started_at"`
}

type chatMessage struct {
	ID        string `json:"id"`
	FromID    string `json:"from_id"`
	FromName  string `json:"from_name"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
	Direction string `json:"direction"`
	Delivery  string `json:"delivery"`
	Error     string `json:"error,omitempty"`
}

type chatState struct {
	mu         sync.RWMutex
	messages   []chatMessage
	peerOnline bool
	peerName   string
}

func setupLogger() *zerolog.Logger {
	logDir := filepath.Join(os.TempDir(), "velo-demo-im")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var writer io.Writer
	if err != nil {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		writer = io.MultiWriter(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}, logFile)
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	return &logger
}

func fatal(logger *zerolog.Logger, msg string) {
	logger.Error().Msg(msg)
	veloerr.ShowErrorDialog(msg)
	os.Exit(1)
}

func main() {
	logger := setupLogger()
	logger.Info().Msgf("Version: %s, OS: %s/%s", Version, runtime.GOOS, runtime.GOARCH)

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	dllPath := filepath.Join(exeDir, "WebView2Loader.dll")
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		if err := os.WriteFile(dllPath, webview2LoaderDLL, 0644); err != nil {
			logger.Warn().Err(err).Msg("failed to extract WebView2Loader.dll")
		} else {
			logger.Info().Msg("extracted WebView2Loader.dll")
		}
	}

	cfg, listener, err := resolveRuntimeConfig()
	if err != nil {
		fatal(logger, "failed to start peer listener: "+err.Error())
	}
	appMode, err := parseVeloMode(*veloModeFlag)
	if err != nil {
		fatal(logger, err.Error())
	}
	if *wsSelfTestFlag {
		appMode = velo.ModeHttp
	}
	logger.Info().
		Str("role", cfg.Role).
		Str("name", cfg.DisplayName).
		Int("listen", cfg.ListenPort).
		Int("peer", cfg.PeerPort).
		Str("velo_mode", appMode.String()).
		Msg("demo-im runtime ready")

	quitOnLastWindowClosed := true
	opt := velo.VeloAppOpt{
		Mode:                   appMode,
		IconData:               appIcon,
		QuitOnLastWindowClosed: &quitOnLastWindowClosed,
	}
	b := velo.NewApp(&opt)
	state := &chatState{}

	startPeerServer(listener, cfg, state, b, logger)
	startPeerMonitor(context.Background(), cfg, state, b, logger)

	registerRoutes(b, cfg, state, logger, appMode)

	b.NewWebview(&velo.VeloWebviewOpt{
		Name:       "demo-im-" + cfg.Role,
		Title:      "Demo IM",
		FrontendFS: frontendFolder,
		Pathname:   "/im",
		Width:      880,
		Height:     620,
		Frameless:  false,
		Hidden:     false,
	})

	if *wsSelfTestFlag {
		if err := runWSTransportSelfTest(b, logger); err != nil {
			fatal(logger, "ws transport self-test failed: "+err.Error())
		}
		logger.Info().Msg("ws transport self-test passed")
		return
	}

	b.Run()
}

func parseVeloMode(value string) (velo.Mode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "bridge":
		return velo.ModeBridge, nil
	case "bridge-http", "bridge_http", "bridgehttp":
		return velo.ModeBridgeHttp, nil
	case "http":
		return velo.ModeHttp, nil
	default:
		return velo.ModeBridge, fmt.Errorf("unknown velo mode %q", value)
	}
}

func resolveRuntimeConfig() (*runtimeConfig, net.Listener, error) {
	listenFlag := flag.Int("listen", 0, "local peer listen port")
	peerFlag := flag.Int("peer", 0, "peer listen port")
	nameFlag := flag.String("name", "", "display name")
	flag.Parse()

	listenPort := firstPort(*listenFlag, os.Getenv("VELO_IM_LISTEN"))
	peerPort := firstPort(*peerFlag, os.Getenv("VELO_IM_PEER"))
	displayName := firstText(*nameFlag, os.Getenv("VELO_IM_NAME"))

	if listenPort > 0 {
		listener, err := bindLocalPort(listenPort)
		if err != nil {
			return nil, nil, err
		}
		if peerPort == 0 {
			peerPort = defaultPeerPort(listenPort)
		}
		role := roleForPort(listenPort)
		if displayName == "" {
			displayName = defaultName(role)
		}
		return newRuntimeConfig(role, displayName, listenPort, peerPort), listener, nil
	}

	listener, err := bindLocalPort(defaultPortA)
	if err == nil {
		role := "a"
		if displayName == "" {
			displayName = defaultName(role)
		}
		return newRuntimeConfig(role, displayName, defaultPortA, defaultPortB), listener, nil
	}

	listener, err = bindLocalPort(defaultPortB)
	if err == nil {
		role := "b"
		if displayName == "" {
			displayName = defaultName(role)
		}
		return newRuntimeConfig(role, displayName, defaultPortB, defaultPortA), listener, nil
	}

	return nil, nil, fmt.Errorf("ports %d and %d are both unavailable", defaultPortA, defaultPortB)
}

func bindLocalPort(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

func firstPort(flagValue int, envValue string) int {
	if flagValue > 0 {
		return flagValue
	}
	if envValue == "" {
		return 0
	}
	port, err := strconv.Atoi(envValue)
	if err != nil || port <= 0 {
		return 0
	}
	return port
}

func firstText(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultPeerPort(listenPort int) int {
	if listenPort == defaultPortA {
		return defaultPortB
	}
	if listenPort == defaultPortB {
		return defaultPortA
	}
	return defaultPortA
}

func roleForPort(port int) string {
	if port == defaultPortB {
		return "b"
	}
	return "a"
}

func defaultName(role string) string {
	if role == "b" {
		return "Bob"
	}
	return "Alice"
}

func newRuntimeConfig(role, displayName string, listenPort, peerPort int) *runtimeConfig {
	return &runtimeConfig{
		InstanceID:  newID("peer"),
		DisplayName: displayName,
		Role:        role,
		ListenPort:  listenPort,
		PeerPort:    peerPort,
		ListenURL:   fmt.Sprintf("http://127.0.0.1:%d", listenPort),
		PeerURL:     fmt.Sprintf("http://127.0.0.1:%d", peerPort),
		StartedAt:   time.Now().Format(time.RFC3339),
	}
}

func registerRoutes(b *velo.Box, cfg *runtimeConfig, state *chatState, logger *zerolog.Logger, appMode velo.Mode) {
	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		messages, peerOnline, peerName := state.Snapshot()
		return c.Ok(velo.H{
			"version":     Version,
			"config":      cfg,
			"messages":    messages,
			"peer_online": peerOnline,
			"peer_name":   peerName,
		})
	})

	b.Get("/api/messages", func(c *velo.BoxContext) interface{} {
		messages, peerOnline, peerName := state.Snapshot()
		return c.Ok(velo.H{
			"messages":    messages,
			"peer_online": peerOnline,
			"peer_name":   peerName,
		})
	})

	b.Get("/api/peer/status", func(c *velo.BoxContext) interface{} {
		online, name := checkPeer(cfg)
		state.SetPeer(online, name)
		return c.Ok(velo.H{"online": online, "name": name})
	})

	b.Get("/api/window/show", func(c *velo.BoxContext) interface{} {
		if appMode != velo.ModeHttp && b.Webview != nil {
			b.Webview.Show()
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/message/send", func(c *velo.BoxContext) interface{} {
		var req struct {
			Text string `json:"text"`
		}
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		text := strings.TrimSpace(req.Text)
		if text == "" {
			return c.Error("message text is required")
		}

		msg := chatMessage{
			ID:        newID("msg"),
			FromID:    cfg.InstanceID,
			FromName:  cfg.DisplayName,
			Text:      text,
			CreatedAt: time.Now().Format(time.RFC3339Nano),
			Direction: "outgoing",
			Delivery:  "sending",
		}
		state.UpsertMessage(msg)
		b.SendMessage(velo.H{"type": "message_sent", "message": msg})

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
			defer cancel()

			if err := postPeerMessage(ctx, cfg, msg); err != nil {
				logger.Warn().Err(err).Str("message_id", msg.ID).Msg("failed to deliver message")
				updated := state.UpdateDelivery(msg.ID, "failed", err.Error())
				state.SetPeer(false, "")
				b.SendMessage(velo.H{"type": "message_delivery", "message": updated})
				b.SendMessage(velo.H{"type": "peer_status", "online": false, "name": ""})
				return
			}

			updated := state.UpdateDelivery(msg.ID, "delivered", "")
			state.SetPeer(true, "")
			b.SendMessage(velo.H{"type": "message_delivery", "message": updated})
			b.SendMessage(velo.H{"type": "peer_status", "online": true})
		}()

		return c.Ok(velo.H{"message": msg})
	})
}

func startPeerServer(listener net.Listener, cfg *runtimeConfig, state *chatState, b *velo.Box, logger *zerolog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/peer/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writePeerJSON(w, http.StatusOK, map[string]interface{}{
			"ok":          true,
			"instance_id": cfg.InstanceID,
			"name":        cfg.DisplayName,
			"role":        cfg.Role,
			"started_at":  cfg.StartedAt,
		})
	})
	mux.HandleFunc("/peer/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var msg chatMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg.Text = strings.TrimSpace(msg.Text)
		if msg.ID == "" || msg.Text == "" {
			http.Error(w, "message id and text are required", http.StatusBadRequest)
			return
		}
		if msg.FromName == "" {
			msg.FromName = "Peer"
		}
		if msg.CreatedAt == "" {
			msg.CreatedAt = time.Now().Format(time.RFC3339Nano)
		}
		msg.Direction = "incoming"
		msg.Delivery = "received"
		msg.Error = ""

		state.UpsertMessage(msg)
		state.SetPeer(true, msg.FromName)
		b.SendMessage(velo.H{"type": "message_received", "message": msg})
		b.SendMessage(velo.H{"type": "peer_status", "online": true, "name": msg.FromName})
		writePeerJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
	})

	server := &http.Server{Handler: mux}
	go func() {
		logger.Info().Str("addr", listener.Addr().String()).Msg("peer server listening")
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("peer server stopped")
		}
	}()
}

func startPeerMonitor(ctx context.Context, cfg *runtimeConfig, state *chatState, b *velo.Box, logger *zerolog.Logger) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			online, name := checkPeer(cfg)
			if state.SetPeer(online, name) {
				logger.Info().Bool("online", online).Str("peer_name", name).Msg("peer status changed")
				b.SendMessage(velo.H{"type": "peer_status", "online": online, "name": name})
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func postPeerMessage(ctx context.Context, cfg *runtimeConfig, msg chatMessage) error {
	outgoing := msg
	outgoing.Direction = "incoming"
	outgoing.Delivery = "received"

	body, err := json.Marshal(outgoing)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.PeerURL+"/peer/message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("peer returned %s: %s", resp.Status, msg)
	}
	return nil
}

func checkPeer(cfg *runtimeConfig) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.PeerURL+"/peer/health", nil)
	if err != nil {
		return false, ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, ""
	}

	var data struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return true, ""
	}
	return true, data.Name
}

func writePeerJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *chatState) Snapshot() ([]chatMessage, bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	messages := append([]chatMessage(nil), s.messages...)
	return messages, s.peerOnline, s.peerName
}

func (s *chatState) UpsertMessage(msg chatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages {
		if s.messages[i].ID == msg.ID {
			s.messages[i] = msg
			return
		}
	}
	s.messages = append(s.messages, msg)
}

func (s *chatState) UpdateDelivery(id, delivery, errText string) chatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages {
		if s.messages[i].ID == id {
			s.messages[i].Delivery = delivery
			s.messages[i].Error = errText
			return s.messages[i]
		}
	}
	return chatMessage{ID: id, Delivery: delivery, Error: errText}
}

func (s *chatState) SetPeer(online bool, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		name = s.peerName
	}
	changed := s.peerOnline != online || s.peerName != name
	s.peerOnline = online
	s.peerName = name
	return changed
}

func newID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf)
}

const (
	selfTestWSOpcodeText = 0x1
)

type selfTestWSClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func runWSTransportSelfTest(b *velo.Box, logger *zerolog.Logger) error {
	go b.Run()

	client, err := dialVeloWSWithRetry(5 * time.Second)
	if err != nil {
		return err
	}
	defer client.close()

	if err := client.sendInvoke("self-app", "/api/app", map[string]interface{}{}); err != nil {
		return err
	}
	if err := client.waitForCallback("self-app", 3*time.Second); err != nil {
		return err
	}
	logger.Info().Msg("ws self-test invoke /api/app passed")

	if err := client.sendInvoke("self-send", "/api/message/send", map[string]interface{}{
		"text": "ws self-test " + time.Now().Format(time.RFC3339Nano),
	}); err != nil {
		return err
	}
	if err := client.waitForCallbackAndMessage("self-send", "message_sent", 5*time.Second); err != nil {
		return err
	}
	logger.Info().Msg("ws self-test invoke /api/message/send and SendMessage push passed")
	return nil
}

func dialVeloWSWithRetry(timeout time.Duration) (*selfTestWSClient, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := dialVeloWS()
		if err == nil {
			return client, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	return nil, fmt.Errorf("connect to ws://127.0.0.1:8080%s: %w", velo.VeloWebSocketPath, lastErr)
}

func dialVeloWS() (*selfTestWSClient, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		conn.Close()
		return nil, err
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: 127.0.0.1:8080\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n\r\n", velo.VeloWebSocketPath, key)
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.Contains(status, "101") {
		conn.Close()
		return nil, fmt.Errorf("websocket handshake returned %s", strings.TrimSpace(status))
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, err
		}
		if line == "\r\n" {
			break
		}
	}

	return &selfTestWSClient{conn: conn, reader: reader}, nil
}

func (c *selfTestWSClient) sendInvoke(id, method string, args interface{}) error {
	payload, err := json.Marshal(map[string]interface{}{
		"id":      id,
		"method":  method,
		"headers": map[string][]string{"Content-Type": {"application/json"}},
		"args":    args,
	})
	if err != nil {
		return err
	}
	return writeSelfTestWSFrame(c.conn, selfTestWSOpcodeText, payload)
}

func (c *selfTestWSClient) waitForCallback(id string, timeout time.Duration) error {
	return c.waitForCallbackAndMessage(id, "", timeout)
}

func (c *selfTestWSClient) waitForCallbackAndMessage(id, messageType string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	gotCallback := false
	gotMessage := messageType == ""

	for time.Now().Before(deadline) {
		if err := c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			return err
		}
		opcode, payload, err := readSelfTestWSFrame(c.reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return err
		}
		if opcode != selfTestWSOpcodeText {
			continue
		}

		var packet struct {
			Type    string          `json:"type"`
			ID      string          `json:"id"`
			Result  json.RawMessage `json:"result"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(payload, &packet); err != nil {
			return err
		}

		if packet.Type == "__velo_callback" && packet.ID == id {
			var result velo.BoxResult
			if err := json.Unmarshal(packet.Result, &result); err != nil {
				return err
			}
			if result.Code != 0 {
				return fmt.Errorf("callback %s failed: %s", id, result.Msg)
			}
			gotCallback = true
		}

		if packet.Type == "__velo_message" && messageType != "" {
			var msg struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(packet.Payload, &msg); err != nil {
				return err
			}
			if msg.Type == messageType {
				gotMessage = true
			}
		}

		if gotCallback && gotMessage {
			return nil
		}
	}

	return fmt.Errorf("timed out waiting for callback=%v message=%v", gotCallback, gotMessage)
}

func (c *selfTestWSClient) close() {
	_ = c.conn.Close()
}

func readSelfTestWSFrame(r *bufio.Reader) (byte, []byte, error) {
	first, err := r.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	second, err := r.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	opcode := first & 0x0f
	length := uint64(second & 0x7f)
	switch length {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf[:]))
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(buf[:])
	}

	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}

func writeSelfTestWSFrame(w io.Writer, opcode byte, payload []byte) error {
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
