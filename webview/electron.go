package webview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type electronBackend struct {
	mu            sync.Mutex
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	configDir     string
	controlServer *http.Server
	controlURL    string
	windows       map[string]*BoxWebviewOptions
	states        map[string]electronWindowState
}

type electronWindowState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type electronAppConfig struct {
	AppName                string                 `json:"app_name"`
	ControlURL             string                 `json:"control_url"`
	HTTPBase               string                 `json:"http_base"`
	QuitOnLastWindowClosed bool                   `json:"quit_on_last_window_closed"`
	Windows                []electronWindowConfig `json:"windows"`
}

type electronWindowConfig struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	URL                  string `json:"url"`
	Pathname             string `json:"pathname"`
	Title                string `json:"title"`
	Width                int    `json:"width"`
	Height               int    `json:"height"`
	X                    int    `json:"x"`
	Y                    int    `json:"y"`
	HasPosition          bool   `json:"has_position"`
	Frameless            bool   `json:"frameless"`
	Hidden               bool   `json:"hidden"`
	HideTrafficLights    bool   `json:"hide_traffic_lights"`
	NonActivating        bool   `json:"non_activating"`
	PreserveStateOnFocus bool   `json:"preserve_state_on_focus"`
	RuntimeJSON          string `json:"runtime_json"`
}

func newElectronBackend() *electronBackend {
	return &electronBackend{
		windows: make(map[string]*BoxWebviewOptions),
		states:  make(map[string]electronWindowState),
	}
}

func (b *electronBackend) OpenWebview(opts *BoxWebviewOptions) *Webview {
	if err := b.start(opts); err != nil {
		fmt.Fprintf(os.Stderr, "[velo] electron: %v\n", err)
		return nil
	}
	err := b.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[velo] electron exited: %v\n", err)
	}
	return nil
}

func (b *electronBackend) OpenWindow(opts *BoxWebviewOptions) *Webview {
	name := normalizeWindowName(opts.Name)
	b.registerWindow(opts)
	if !b.running() {
		if err := b.start(opts); err != nil {
			fmt.Fprintf(os.Stderr, "[velo] electron: %v\n", err)
			return nil
		}
		return NewHandle(name, EngineElectron)
	}
	if err := b.sendCommand(electronCommand{
		Type:   "open_window",
		Window: newElectronWindowConfig(opts),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[velo] electron open window: %v\n", err)
	}
	return NewHandle(name, EngineElectron)
}

func (b *electronBackend) FocusWindow(opts *BoxWebviewOptions) bool {
	name := normalizeWindowName(opts.Name)
	if !b.running() {
		return false
	}
	b.mu.Lock()
	_, exists := b.windows[name]
	b.mu.Unlock()
	if !exists {
		return false
	}
	if err := b.sendCommand(electronCommand{
		Type:   "focus_window",
		Name:   name,
		Window: newElectronWindowConfig(opts),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[velo] electron focus window: %v\n", err)
		return false
	}
	b.registerWindow(opts)
	return true
}

func (b *electronBackend) SendCallback(id, result string) {}

func (b *electronBackend) SendMessage(payload string) bool {
	return false
}

func (b *electronBackend) SetTitle(name, title string) {
	b.windowControl(name, "set_title", map[string]interface{}{"title": title})
}

func (b *electronBackend) SetSize(name string, width, height int) {
	b.windowControl(name, "__velo/window/set_size", map[string]interface{}{"width": width, "height": height})
}

func (b *electronBackend) SetMinSize(name string, width, height int) {
	b.windowControl(name, "set_min_size", map[string]interface{}{"width": width, "height": height})
}

func (b *electronBackend) SetMaxSize(name string, width, height int) {
	b.windowControl(name, "set_max_size", map[string]interface{}{"width": width, "height": height})
}

func (b *electronBackend) SetPosition(name string, x, y int) {
	b.windowControl(name, "set_position", map[string]interface{}{"x": x, "y": y})
}

func (b *electronBackend) GetPosition(name string) (int, int) {
	state := b.state(name)
	return state.X, state.Y
}

func (b *electronBackend) GetSize(name string) (int, int) {
	state := b.state(name)
	return state.Width, state.Height
}

func (b *electronBackend) Show(name string) {
	b.windowControl(name, "show", nil)
}

func (b *electronBackend) Hide(name string) {
	b.windowControl(name, "__velo/window/hide", nil)
}

func (b *electronBackend) Minimize(name string) {
	b.windowControl(name, "__velo/window/minimize", nil)
}

func (b *electronBackend) Maximize(name string) {
	b.windowControl(name, "__velo/window/maximize", nil)
}

func (b *electronBackend) Fullscreen(name string) {
	b.windowControl(name, "fullscreen", nil)
}

func (b *electronBackend) UnFullscreen(name string) {
	b.windowControl(name, "unfullscreen", nil)
}

func (b *electronBackend) Restore(name string) {
	b.windowControl(name, "__velo/window/restore", nil)
}

func (b *electronBackend) SetAlwaysOnTop(name string, onTop bool) {
	b.windowControl(name, "__velo/window/set_always_on_top", map[string]interface{}{"onTop": onTop})
}

func (b *electronBackend) SetURL(name, targetURL string) {
	b.windowControl(name, "set_url", map[string]interface{}{"url": targetURL})
}

func (b *electronBackend) Close(name string) {
	b.windowControl(name, "__velo/window/close", nil)
}

func (b *electronBackend) windowControl(name, method string, args interface{}) {
	if !b.running() {
		return
	}
	if err := b.sendCommand(electronCommand{
		Type:   "window_control",
		Name:   normalizeWindowName(name),
		Method: method,
		Args:   args,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[velo] electron window control %s: %v\n", method, err)
	}
}

func (b *electronBackend) running() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cmd != nil && b.stdin != nil
}

func (b *electronBackend) state(name string) electronWindowState {
	name = normalizeWindowName(name)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.states[name]
}

func (b *electronBackend) registerWindow(opts *BoxWebviewOptions) {
	if opts == nil {
		return
	}
	name := normalizeWindowName(opts.Name)
	b.mu.Lock()
	b.windows[name] = opts
	if opts.Width > 0 || opts.Height > 0 {
		state := b.states[name]
		if opts.Width > 0 {
			state.Width = opts.Width
		}
		if opts.Height > 0 {
			state.Height = opts.Height
		}
		if opts.HasPosition {
			state.X = opts.X
			state.Y = opts.Y
		}
		b.states[name] = state
	}
	b.mu.Unlock()
}

func (b *electronBackend) start(opts *BoxWebviewOptions) error {
	if opts == nil {
		return errors.New("missing electron webview options")
	}
	b.registerWindow(opts)

	b.mu.Lock()
	if b.cmd != nil {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	if err := b.ensureControlServer(); err != nil {
		return err
	}

	configDir, err := os.MkdirTemp("", "velo-electron-*")
	if err != nil {
		return err
	}
	if err := writeElectronApp(configDir); err != nil {
		os.RemoveAll(configDir)
		return err
	}

	appConfig := electronAppConfig{
		AppName:                opts.AppName,
		ControlURL:             b.controlURL,
		HTTPBase:               electronHTTPBase(opts.URL),
		QuitOnLastWindowClosed: opts.QuitOnLastWindowClosed,
		Windows:                []electronWindowConfig{newElectronWindowConfig(opts)},
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := writeJSON(configPath, appConfig); err != nil {
		os.RemoveAll(configDir)
		return err
	}

	command, args, err := electronCommandPath(opts.ElectronCommand)
	if err != nil {
		os.RemoveAll(configDir)
		return err
	}
	args = append(args, configDir, "--velo-electron-config", configPath)
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir, _ = os.Getwd()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.RemoveAll(configDir)
		return err
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(configDir)
		return err
	}

	b.mu.Lock()
	b.cmd = cmd
	b.stdin = stdin
	b.configDir = configDir
	b.mu.Unlock()
	return nil
}

func (b *electronBackend) wait() error {
	b.mu.Lock()
	cmd := b.cmd
	configDir := b.configDir
	b.mu.Unlock()
	if cmd == nil {
		return nil
	}
	err := cmd.Wait()
	b.mu.Lock()
	if b.cmd == cmd {
		b.cmd = nil
		b.stdin = nil
		b.configDir = ""
		b.windows = make(map[string]*BoxWebviewOptions)
		b.states = make(map[string]electronWindowState)
	}
	b.mu.Unlock()
	if configDir != "" {
		os.RemoveAll(configDir)
	}
	return err
}

type electronCommand struct {
	Type   string               `json:"type"`
	Name   string               `json:"name,omitempty"`
	Method string               `json:"method,omitempty"`
	Args   interface{}          `json:"args,omitempty"`
	Window electronWindowConfig `json:"window,omitempty"`
}

func (b *electronBackend) sendCommand(command electronCommand) error {
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stdin == nil {
		return errors.New("electron stdin is unavailable")
	}
	_, err = b.stdin.Write(append(data, '\n'))
	return err
}

func (b *electronBackend) ensureControlServer() error {
	b.mu.Lock()
	if b.controlServer != nil {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/event", b.handleEvent)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	server := &http.Server{Handler: mux}
	controlURL := "http://" + ln.Addr().String()

	b.mu.Lock()
	if b.controlServer != nil {
		b.mu.Unlock()
		ln.Close()
		return nil
	}
	b.controlServer = server
	b.controlURL = controlURL
	b.mu.Unlock()

	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "[velo] electron control server: %v\n", err)
		}
	}()
	return nil
}

func (b *electronBackend) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var event struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Event   string `json:"event"`
		Payload string `json:"payload"`
		X       int    `json:"x"`
		Y       int    `json:"y"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := normalizeWindowName(event.Name)
	switch event.Type {
	case "window_state":
		b.mu.Lock()
		b.states[name] = electronWindowState{X: event.X, Y: event.Y, Width: event.Width, Height: event.Height}
		b.mu.Unlock()
	case "drag_drop":
		if opts := b.windowOptions(name); opts != nil && opts.HandleDragDrop != nil {
			go opts.HandleDragDrop(event.Event, event.Payload)
		}
	case "reopen":
		if opts := b.windowOptions(name); opts != nil && opts.HandleReopen != nil {
			go opts.HandleReopen()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (b *electronBackend) windowOptions(name string) *BoxWebviewOptions {
	name = normalizeWindowName(name)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.windows[name]
}

func newElectronWindowConfig(opts *BoxWebviewOptions) electronWindowConfig {
	if opts == nil {
		return electronWindowConfig{Name: "default"}
	}
	return electronWindowConfig{
		ID:                   opts.ID,
		Name:                 normalizeWindowName(opts.Name),
		URL:                  opts.URL,
		Pathname:             opts.Pathname,
		Title:                opts.Title,
		Width:                opts.Width,
		Height:               opts.Height,
		X:                    opts.X,
		Y:                    opts.Y,
		HasPosition:          opts.HasPosition,
		Frameless:            opts.Frameless,
		Hidden:               opts.Hidden,
		HideTrafficLights:    opts.HideTrafficLights,
		NonActivating:        opts.NonActivating,
		PreserveStateOnFocus: opts.PreserveStateOnFocus,
		RuntimeJSON:          opts.RuntimeJSON,
	}
}

func normalizeWindowName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func electronHTTPBase(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return u.Scheme + "://" + u.Host
	}
	return "http://127.0.0.1:8080"
}

func electronCommandPath(configured string) (string, []string, error) {
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured, nil, nil
	}
	if override := strings.TrimSpace(os.Getenv("VELO_ELECTRON_BIN")); override != "" {
		return override, nil, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates := []string{filepath.Join(cwd, "node_modules", ".bin", "electron")}
		if runtime.GOOS == "windows" {
			candidates = append(candidates, filepath.Join(cwd, "node_modules", ".bin", "electron.cmd"))
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil, nil
			}
		}
	}
	if path, err := exec.LookPath("electron"); err == nil {
		return path, nil, nil
	}
	return "", nil, errors.New("electron binary not found; install electron in node_modules or set VELO_ELECTRON_BIN")
}

func writeElectronApp(dir string) error {
	files := map[string]string{
		"package.json": electronPackageJSON,
		"main.js":      electronMainJS,
		"preload.js":   electronPreloadJS,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(path string, value interface{}) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

const electronPackageJSON = `{"name":"velo-electron-host","version":"0.0.0","private":true,"main":"main.js"}`

const electronMainJS = `
const { app, BrowserWindow, ipcMain, protocol } = require("electron");
const fs = require("fs");
const path = require("path");
const readline = require("readline");

protocol.registerSchemesAsPrivileged([
  {
    scheme: "velo",
    privileges: {
      standard: true,
      secure: true,
      supportFetchAPI: true,
      corsEnabled: true,
      stream: true
    }
  }
]);

function argValue(name) {
  for (let i = 0; i < process.argv.length; i++) {
    if (process.argv[i] === name && i + 1 < process.argv.length) {
      return process.argv[i + 1];
    }
    if (process.argv[i].startsWith(name + "=")) {
      return process.argv[i].slice(name.length + 1);
    }
  }
  return "";
}

const configPath = argValue("--velo-electron-config");
if (!configPath) {
  throw new Error("missing --velo-electron-config");
}
const configDir = path.dirname(configPath);
const config = JSON.parse(fs.readFileSync(configPath, "utf8"));
const preloadPath = path.join(configDir, "preload.js");
const windowsByName = new Map();
const namesByWebContents = new Map();

function safeName(name) {
  return String(name || "default").replace(/[^a-zA-Z0-9_.-]/g, "_");
}

function runtimeInfoForWindow(windowConfig) {
  try {
    return JSON.parse(windowConfig.runtime_json || "{}");
  } catch (_) {
    return {};
  }
}

function wsURLForWindow(windowConfig) {
  try {
    const url = new URL(windowConfig.url || config.http_base || "http://127.0.0.1:8080");
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = "/__velo/ws";
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch (_) {
    return "ws://127.0.0.1:8080/__velo/ws";
  }
}

function writeRuntimeFile(windowConfig) {
  const runtimePath = path.join(configDir, "runtime-" + safeName(windowConfig.id || windowConfig.name) + ".json");
  fs.writeFileSync(runtimePath, JSON.stringify({
    runtime_info: runtimeInfoForWindow(windowConfig),
    ws_url: wsURLForWindow(windowConfig),
    window_name: windowConfig.name || "default"
  }));
  return runtimePath;
}

function postEvent(payload) {
  if (!config.control_url || typeof fetch !== "function") {
    return;
  }
  fetch(config.control_url + "/event", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload)
  }).catch(() => {});
}

function postWindowState(name, win) {
  if (!win || win.isDestroyed()) {
    return;
  }
  const bounds = win.getBounds();
  postEvent({
    type: "window_state",
    name,
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height
  });
}

function createWindow(windowConfig) {
  const name = windowConfig.name || "default";
  const existing = windowsByName.get(name);
  if (existing && !existing.isDestroyed()) {
    if (windowConfig.title) {
      existing.setTitle(windowConfig.title);
    }
    if (windowConfig.url && !windowConfig.preserve_state_on_focus) {
      existing.loadURL(windowConfig.url);
    }
    existing.show();
    existing.focus();
    postWindowState(name, existing);
    return existing;
  }

  const runtimePath = writeRuntimeFile(windowConfig);
  const options = {
    title: windowConfig.title || config.app_name || "Velo",
    width: windowConfig.width || 1024,
    height: windowConfig.height || 768,
    show: !windowConfig.hidden,
    frame: !windowConfig.frameless,
    webPreferences: {
      preload: preloadPath,
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
      additionalArguments: ["--velo-runtime-config=" + runtimePath]
    }
  };
  if (windowConfig.has_position) {
    options.x = windowConfig.x;
    options.y = windowConfig.y;
  }
  if (windowConfig.hide_traffic_lights) {
    options.titleBarStyle = "hidden";
  }
  if (windowConfig.non_activating) {
    options.focusable = false;
  }

  const win = new BrowserWindow(options);
  windowsByName.set(name, win);
  namesByWebContents.set(win.webContents.id, name);

  let stateTimer = null;
  const scheduleState = () => {
    if (stateTimer) {
      clearTimeout(stateTimer);
    }
    stateTimer = setTimeout(() => postWindowState(name, win), 80);
  };
  win.on("resize", scheduleState);
  win.on("move", scheduleState);
  win.on("close", () => postWindowState(name, win));
  win.on("closed", () => {
    windowsByName.delete(name);
    namesByWebContents.delete(win.webContents.id);
  });
  win.webContents.on("did-finish-load", () => postWindowState(name, win));
  win.loadURL(windowConfig.url || config.http_base || "about:blank");
  return win;
}

function windowForName(name) {
  const win = windowsByName.get(name || "default");
  if (!win || win.isDestroyed()) {
    return null;
  }
  return win;
}

function handleWindowControl(win, method, args) {
  if (!win || win.isDestroyed()) {
    return { success: false };
  }
  args = args || {};
  switch (method) {
    case "set_title":
      win.setTitle(String(args.title || ""));
      break;
    case "__velo/window/start_drag":
      break;
    case "__velo/window/close":
      win.close();
      return { success: true };
    case "__velo/window/minimize":
      win.minimize();
      break;
    case "__velo/window/hide":
      win.hide();
      break;
    case "show":
      win.show();
      win.focus();
      break;
    case "__velo/window/set_size":
      if (args.width > 0 && args.height > 0) {
        win.setSize(Math.floor(args.width), Math.floor(args.height));
      }
      break;
    case "set_min_size":
      if (args.width > 0 && args.height > 0) {
        win.setMinimumSize(Math.floor(args.width), Math.floor(args.height));
      }
      break;
    case "set_max_size":
      if (args.width > 0 && args.height > 0) {
        win.setMaximumSize(Math.floor(args.width), Math.floor(args.height));
      }
      break;
    case "set_position":
      win.setPosition(Math.floor(args.x || 0), Math.floor(args.y || 0));
      break;
    case "__velo/window/state": {
      const bounds = win.getBounds();
      return { success: true, x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height };
    }
    case "__velo/window/toggle_maximize":
      if (win.isMaximized()) {
        win.unmaximize();
      } else {
        win.maximize();
      }
      break;
    case "__velo/window/maximize":
      win.maximize();
      break;
    case "__velo/window/restore":
      if (win.isMinimized()) {
        win.restore();
      }
      if (win.isMaximized()) {
        win.unmaximize();
      }
      win.show();
      break;
    case "__velo/window/set_always_on_top":
      win.setAlwaysOnTop(!!args.onTop);
      break;
    case "fullscreen":
      win.setFullScreen(true);
      break;
    case "unfullscreen":
      win.setFullScreen(false);
      break;
    case "set_url":
      if (args.url) {
        win.loadURL(String(args.url));
      }
      break;
    default:
      return { success: false };
  }
  return { success: true };
}

ipcMain.handle("velo-window-control", (event, payload) => {
  const name = namesByWebContents.get(event.sender.id) || "default";
  return handleWindowControl(windowForName(name), payload && payload.method, payload && payload.args);
});

ipcMain.on("velo-drag-drop", (event, payload) => {
  const name = namesByWebContents.get(event.sender.id) || "default";
  postEvent({
    type: "drag_drop",
    name,
    event: payload && payload.event || "drop",
    payload: JSON.stringify(payload && payload.payload || {})
  });
});

app.on("window-all-closed", () => {
  if (config.quit_on_last_window_closed !== false) {
    app.quit();
  }
});

app.on("activate", () => {
  const first = config.windows && config.windows[0];
  if (BrowserWindow.getAllWindows().length === 0 && first) {
    createWindow(first);
  } else if (first) {
    const win = windowForName(first.name || "default");
    if (win) {
      win.show();
      win.focus();
    }
  }
  if (first) {
    postEvent({ type: "reopen", name: first.name || "default" });
  }
});

app.whenReady().then(() => {
  protocol.handle("velo", async (request) => {
    const requestURL = new URL(request.url);
    const target = (config.http_base || "http://127.0.0.1:8080") + requestURL.pathname + requestURL.search;
    const init = { method: request.method, headers: request.headers };
    if (request.method !== "GET" && request.method !== "HEAD") {
      init.body = Buffer.from(await request.arrayBuffer());
    }
    return fetch(target, init);
  });

  for (const windowConfig of config.windows || []) {
    createWindow(windowConfig);
  }

  const rl = readline.createInterface({ input: process.stdin });
  rl.on("line", (line) => {
    if (!line.trim()) {
      return;
    }
    let command = null;
    try {
      command = JSON.parse(line);
    } catch (_) {
      return;
    }
    if (command.type === "open_window") {
      createWindow(command.window || {});
      return;
    }
    if (command.type === "focus_window") {
      createWindow(command.window || { name: command.name || "default" });
      return;
    }
    if (command.type === "window_control") {
      handleWindowControl(windowForName(command.name || "default"), command.method, command.args);
    }
  });
});
`

const electronPreloadJS = `
const { contextBridge, ipcRenderer } = require("electron");
const fs = require("fs");

function argValue(name) {
  for (let i = 0; i < process.argv.length; i++) {
    if (process.argv[i].startsWith(name + "=")) {
      return process.argv[i].slice(name.length + 1);
    }
  }
  return "";
}

const runtimeConfigPath = argValue("--velo-runtime-config");
const runtimeConfig = runtimeConfigPath ? JSON.parse(fs.readFileSync(runtimeConfigPath, "utf8")) : {};
const runtimeInfo = runtimeConfig.runtime_info || {};
const wsURL = runtimeConfig.ws_url || "ws://127.0.0.1:8080/__velo/ws";

const callbacks = {};
const messageHandlers = [];
let socket = null;
let connecting = null;

function handlePacket(data) {
  let packet = null;
  try {
    packet = typeof data === "string" ? JSON.parse(data) : data;
  } catch (_) {
    return;
  }
  if (!packet) {
    return;
  }
  if (packet.type === "__velo_callback") {
    const cb = callbacks[packet.id];
    if (cb) {
      delete callbacks[packet.id];
      cb(packet.result);
    }
    return;
  }
  if (packet.type === "__velo_message") {
    for (const handler of messageHandlers.slice()) {
      try {
        handler(packet.payload);
      } catch (_) {}
    }
    return;
  }
  for (const handler of messageHandlers.slice()) {
    try {
      handler(packet);
    } catch (_) {}
  }
}

function ensureSocket() {
  if (socket && socket.readyState === WebSocket.OPEN) {
    return Promise.resolve(socket);
  }
  if (connecting) {
    return connecting;
  }
  connecting = new Promise((resolve, reject) => {
    let settled = false;
    let timer = null;
    function finish(fn, value) {
      if (settled) {
        return;
      }
      settled = true;
      if (timer) {
        clearTimeout(timer);
      }
      connecting = null;
      fn(value);
    }
    let nextSocket = null;
    try {
      nextSocket = new WebSocket(wsURL);
    } catch (error) {
      finish(reject, error);
      return;
    }
    timer = setTimeout(() => {
      try {
        nextSocket.close();
      } catch (_) {}
      finish(reject, new Error("Go WebSocket connection timed out"));
    }, 10000);
    nextSocket.onopen = () => {
      socket = nextSocket;
      finish(resolve, nextSocket);
    };
    nextSocket.onmessage = (event) => handlePacket(event.data);
    nextSocket.onerror = () => finish(reject, new Error("Go WebSocket is not available"));
    nextSocket.onclose = () => {
      if (socket === nextSocket) {
        socket = null;
      }
      finish(reject, new Error("Go WebSocket closed"));
    };
  });
  return connecting;
}

function invoke(method, options) {
  options = options || {};
  if (typeof method === "string" && method.indexOf("__velo/window/") === 0) {
    return ipcRenderer.invoke("velo-window-control", {
      method,
      args: options.args || {}
    });
  }
  return new Promise((resolve, reject) => {
    const id = String(Date.now()) + Math.random().toString(16).slice(2);
    callbacks[id] = (result) => {
      if (typeof result === "string") {
        try {
          resolve(JSON.parse(result));
          return;
        } catch (_) {}
      }
      resolve(result);
    };
    const payload = {
      id,
      method,
      headers: options.headers,
      args: options.args
    };
    ensureSocket().then((ws) => {
      ws.send(JSON.stringify(payload));
    }).catch((error) => {
      delete callbacks[id];
      reject(error);
    });
  });
}

function onGoMessage(handler) {
  if (typeof handler !== "function") {
    return;
  }
  messageHandlers.push(handler);
  ensureSocket().catch(() => {});
}

contextBridge.exposeInMainWorld("__VELO__", runtimeInfo);
contextBridge.exposeInMainWorld("invoke", invoke);
contextBridge.exposeInMainWorld("goCall", invoke);
contextBridge.exposeInMainWorld("onGoMessage", onGoMessage);

window.addEventListener("drop", (event) => {
  const files = [];
  if (event.dataTransfer && event.dataTransfer.files) {
    for (const file of event.dataTransfer.files) {
      if (file && file.path) {
        files.push(file.path);
      }
    }
  }
  if (files.length > 0) {
    ipcRenderer.send("velo-drag-drop", {
      event: "drop",
      payload: { files }
    });
  }
}, true);

ensureSocket().catch(() => {});
`
