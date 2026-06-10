package inputsource

import (
	"context"
	"strings"
	"sync"
	"time"
)

// RuleMode describes how a foreground app rule should resolve.
type RuleMode string

const (
	// RuleUseDefault falls back to Config.DefaultSourceID.
	RuleUseDefault RuleMode = "default"
	// RuleLock pins the app to AppRule.SourceID.
	RuleLock RuleMode = "lock"
	// RuleIgnore disables enforcement while the app is foreground.
	RuleIgnore RuleMode = "ignore"
)

// AppRule configures input-source behavior for one app.
type AppRule struct {
	// AppID matches App.ID. Matching is case-insensitive on Windows and exact
	// on other platforms.
	AppID string
	Mode  RuleMode
	// SourceID is required when Mode is RuleLock.
	SourceID string
}

// Config controls a Manager.
type Config struct {
	Enabled         bool
	DefaultSourceID string
	AppRules        []AppRule
	// PollInterval controls how often the manager checks foreground app/input
	// source state. Zero uses DefaultPollInterval.
	PollInterval time.Duration
	// SuppressionWindow avoids re-selecting repeatedly while the OS is still
	// settling a switch we requested. Zero uses DefaultSuppressionWindow.
	SuppressionWindow time.Duration
}

// ActivationReason explains why a Manager selected a source.
type ActivationReason string

const (
	ActivationAppChanged    ActivationReason = "app_changed"
	ActivationSourceChanged ActivationReason = "source_changed"
	ActivationLockEngaged   ActivationReason = "lock_engaged"
)

// ActivationEvent is emitted after a successful forced source selection.
type ActivationEvent struct {
	Time       time.Time
	App        App
	SourceID   string
	Reason     ActivationReason
	Duration   time.Duration
	PreviousID string
}

const (
	DefaultPollInterval      = 250 * time.Millisecond
	DefaultSuppressionWindow = 300 * time.Millisecond
)

// Manager keeps the active input source pinned to the rule matching the
// foreground app.
type Manager struct {
	mu       sync.RWMutex
	config   Config
	cancel   context.CancelFunc
	done     chan struct{}
	running  bool
	settleTo time.Time

	// OnActivation is called after a successful forced switch.
	OnActivation func(ActivationEvent)
	// OnError is called for non-fatal polling/select errors.
	OnError func(error)
}

// NewManager creates a manager with config.
func NewManager(config Config) *Manager {
	return &Manager{config: normalizeConfig(config)}
}

// SetConfig replaces the manager configuration. It is safe to call while the
// manager is running.
func (m *Manager) SetConfig(config Config) {
	m.mu.Lock()
	m.config = normalizeConfig(config)
	m.settleTo = time.Time{}
	m.mu.Unlock()
}

// Config returns the current manager configuration.
func (m *Manager) Config() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneConfig(m.config)
}

// Start begins enforcing rules until ctx is cancelled or Stop is called.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})
	m.running = true
	m.mu.Unlock()

	go m.loop(runCtx)
	return nil
}

// Stop stops a running manager and waits for its goroutine to exit.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	cancel := m.cancel
	done := m.done
	m.mu.Unlock()

	cancel()
	<-done
}

func (m *Manager) loop(ctx context.Context) {
	defer func() {
		m.mu.Lock()
		m.running = false
		m.cancel = nil
		m.done = nil
		m.mu.Unlock()
	}()
	defer close(m.done)

	var lastAppID string
	var lastSourceID string
	_ = m.enforce(ActivationLockEngaged, &lastAppID, &lastSourceID)

	ticker := time.NewTicker(m.Config().PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reason := ActivationSourceChanged
			app, err := FrontmostApp()
			if err == nil && app.ID != lastAppID {
				reason = ActivationAppChanged
			}
			_ = m.enforce(reason, &lastAppID, &lastSourceID)
		}
	}
}

func (m *Manager) enforce(reason ActivationReason, lastAppID *string, lastSourceID *string) error {
	config := m.Config()
	if !config.Enabled {
		return nil
	}

	app, err := FrontmostApp()
	if err != nil {
		m.reportError(err)
		return err
	}
	*lastAppID = app.ID

	target := resolveTarget(config, app)
	if target == "" {
		return nil
	}

	current, err := Current()
	if err != nil {
		m.reportError(err)
		return err
	}
	if current.ID == target {
		*lastSourceID = current.ID
		return nil
	}

	m.mu.RLock()
	settling := time.Now().Before(m.settleTo)
	m.mu.RUnlock()
	if settling {
		return nil
	}

	start := time.Now()
	if err := Select(target); err != nil {
		m.reportError(err)
		return err
	}

	m.mu.Lock()
	m.settleTo = time.Now().Add(config.SuppressionWindow)
	onActivation := m.OnActivation
	m.mu.Unlock()

	if onActivation != nil {
		onActivation(ActivationEvent{
			Time:       time.Now(),
			App:        app,
			SourceID:   target,
			Reason:     reason,
			Duration:   time.Since(start),
			PreviousID: current.ID,
		})
	}
	*lastSourceID = target
	return nil
}

func (m *Manager) reportError(err error) {
	m.mu.RLock()
	onError := m.OnError
	m.mu.RUnlock()
	if onError != nil {
		onError(err)
	}
}

func normalizeConfig(config Config) Config {
	if config.PollInterval <= 0 {
		config.PollInterval = DefaultPollInterval
	}
	if config.SuppressionWindow <= 0 {
		config.SuppressionWindow = DefaultSuppressionWindow
	}
	config.AppRules = append([]AppRule(nil), config.AppRules...)
	return config
}

func cloneConfig(config Config) Config {
	config.AppRules = append([]AppRule(nil), config.AppRules...)
	return config
}

func resolveTarget(config Config, app App) string {
	for _, rule := range config.AppRules {
		if !sameAppID(rule.AppID, app.ID) {
			continue
		}
		switch rule.Mode {
		case RuleIgnore:
			return ""
		case RuleLock:
			if rule.SourceID != "" {
				return rule.SourceID
			}
			return config.DefaultSourceID
		case RuleUseDefault, "":
			return config.DefaultSourceID
		default:
			return config.DefaultSourceID
		}
	}
	return config.DefaultSourceID
}

func sameAppID(ruleID, appID string) bool {
	if isWindows() {
		return strings.EqualFold(ruleID, appID)
	}
	return ruleID == appID
}
