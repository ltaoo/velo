package desktopapp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/ltaoo/velo/inputsource"
	"github.com/ltaoo/velo/store"
	"github.com/rs/zerolog"
)

const inputSourceLockSettingsKey = "demo-desktop:settings:input-source-lock:v1"

type InputSourceLockSettings struct {
	Enabled         bool                 `json:"enabled"`
	DefaultSourceID string               `json:"defaultSourceId"`
	AppRules        []InputSourceAppRule `json:"appRules"`
}

type InputSourceAppRule struct {
	AppID    string `json:"appId"`
	AppName  string `json:"appName"`
	SourceID string `json:"sourceId"`
	Enabled  bool   `json:"enabled"`
}

type InputSourceLockService struct {
	mu      sync.Mutex
	manager *inputsource.Manager
	cancel  context.CancelFunc
	logger  *zerolog.Logger
}

func NewInputSourceLockService(logger *zerolog.Logger) *InputSourceLockService {
	return &InputSourceLockService{logger: logger}
}

func (s *InputSourceLockService) Apply(settings InputSourceLockSettings) {
	settings = normalizeInputSourceLockSettings(settings)
	config := inputSourceManagerConfig(settings)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.manager == nil {
		s.manager = inputsource.NewManager(config)
		s.manager.OnActivation = func(event inputsource.ActivationEvent) {
			if s.logger == nil {
				return
			}
			s.logger.Info().
				Str("app", event.App.ID).
				Str("source", event.SourceID).
				Str("reason", string(event.Reason)).
				Dur("duration", event.Duration).
				Msg("input source lock activated")
		}
		s.manager.OnError = func(err error) {
			if s.logger != nil {
				s.logger.Warn().Err(err).Msg("input source lock error")
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		if err := s.manager.Start(ctx); err != nil && s.logger != nil {
			s.logger.Warn().Err(err).Msg("failed to start input source lock manager")
		}
		return
	}
	s.manager.SetConfig(config)
}

func (s *InputSourceLockService) Stop() {
	s.mu.Lock()
	manager := s.manager
	cancel := s.cancel
	s.manager = nil
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if manager != nil {
		manager.Stop()
	}
}

func loadStoredInputSourceLockSettings(raw json.RawMessage) (InputSourceLockSettings, error) {
	if raw == nil {
		return defaultInputSourceLockSettings(), nil
	}
	var settings InputSourceLockSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return InputSourceLockSettings{}, fmt.Errorf("read input source lock settings: %w", err)
	}
	return normalizeInputSourceLockSettings(settings), nil
}

func marshalInputSourceLockSettingsForStore(settings InputSourceLockSettings) ([]byte, error) {
	return json.Marshal(normalizeInputSourceLockSettings(settings))
}

func defaultInputSourceLockSettings() InputSourceLockSettings {
	return InputSourceLockSettings{AppRules: []InputSourceAppRule{}}
}

func normalizeInputSourceLockSettings(settings InputSourceLockSettings) InputSourceLockSettings {
	next := InputSourceLockSettings{
		Enabled:         settings.Enabled,
		DefaultSourceID: strings.TrimSpace(settings.DefaultSourceID),
		AppRules:        make([]InputSourceAppRule, 0, len(settings.AppRules)),
	}
	seen := make(map[string]bool)
	for _, rule := range settings.AppRules {
		rule.AppID = strings.TrimSpace(rule.AppID)
		rule.AppName = strings.TrimSpace(rule.AppName)
		rule.SourceID = strings.TrimSpace(rule.SourceID)
		if rule.AppID == "" || seen[rule.AppID] {
			continue
		}
		seen[rule.AppID] = true
		next.AppRules = append(next.AppRules, rule)
	}
	return next
}

func inputSourceManagerConfig(settings InputSourceLockSettings) inputsource.Config {
	settings = normalizeInputSourceLockSettings(settings)
	rules := make([]inputsource.AppRule, 0, len(settings.AppRules))
	for _, rule := range settings.AppRules {
		if !rule.Enabled || rule.SourceID == "" {
			continue
		}
		rules = append(rules, inputsource.AppRule{
			AppID:    rule.AppID,
			Mode:     inputsource.RuleLock,
			SourceID: rule.SourceID,
		})
	}
	return inputsource.Config{
		Enabled:         settings.Enabled,
		DefaultSourceID: settings.DefaultSourceID,
		AppRules:        rules,
	}
}

func applyStoredInputSourceLockSettings(st *store.Store, service *InputSourceLockService, logger *zerolog.Logger) {
	if st == nil || service == nil {
		return
	}
	settings, err := loadStoredInputSourceLockSettings(st.Get(inputSourceLockSettingsKey))
	if err != nil {
		if logger != nil {
			logger.Warn().Err(err).Msg("failed to read input source lock settings")
		}
		return
	}
	service.Apply(settings)
}
