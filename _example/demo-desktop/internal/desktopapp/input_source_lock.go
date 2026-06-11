package desktopapp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

type InputSourceLockMissingRule struct {
	AppID    string `json:"appId"`
	AppName  string `json:"appName"`
	SourceID string `json:"sourceId"`
}

type InputSourceLockAvailability struct {
	HasMissingSources      bool                         `json:"hasMissingSources"`
	MissingDefaultSourceID string                       `json:"missingDefaultSourceId"`
	MissingSourceIDs       []string                     `json:"missingSourceIds"`
	MissingAppRules        []InputSourceLockMissingRule `json:"missingAppRules"`
	RuntimeEnabled         bool                         `json:"runtimeEnabled"`
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
	config := inputSourceManagerConfigForCurrentMachine(settings, s.logger)

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
	return inputSourceManagerConfigForAvailable(settings, nil)
}

func inputSourceManagerConfigForCurrentMachine(settings InputSourceLockSettings, logger *zerolog.Logger) inputsource.Config {
	sources, err := inputsource.List()
	if err != nil {
		if logger != nil {
			logger.Warn().Err(err).Msg("failed to list input sources for lock settings")
		}
		return inputSourceManagerConfigForAvailable(settings, map[string]bool{})
	}
	return inputSourceManagerConfigForAvailable(settings, inputSourceIDSet(sources))
}

func inputSourceManagerConfigForAvailable(settings InputSourceLockSettings, availableSourceIDs map[string]bool) inputsource.Config {
	settings = normalizeInputSourceLockSettings(settings)
	defaultSourceID := settings.DefaultSourceID
	if defaultSourceID != "" && availableSourceIDs != nil && !availableSourceIDs[defaultSourceID] {
		defaultSourceID = ""
	}

	rules := make([]inputsource.AppRule, 0, len(settings.AppRules))
	hasLockTarget := defaultSourceID != ""
	for _, rule := range settings.AppRules {
		if !rule.Enabled || rule.SourceID == "" {
			continue
		}
		if availableSourceIDs != nil && !availableSourceIDs[rule.SourceID] {
			rules = append(rules, inputsource.AppRule{
				AppID: rule.AppID,
				Mode:  inputsource.RuleIgnore,
			})
			continue
		}
		rules = append(rules, inputsource.AppRule{
			AppID:    rule.AppID,
			Mode:     inputsource.RuleLock,
			SourceID: rule.SourceID,
		})
		hasLockTarget = true
	}

	enabled := settings.Enabled
	if availableSourceIDs != nil {
		enabled = enabled && hasLockTarget
	}
	return inputsource.Config{
		Enabled:         enabled,
		DefaultSourceID: defaultSourceID,
		AppRules:        rules,
	}
}

func inputSourceIDSet(sources []inputsource.Source) map[string]bool {
	ids := make(map[string]bool, len(sources))
	for _, source := range sources {
		id := strings.TrimSpace(source.ID)
		if id == "" {
			continue
		}
		ids[id] = true
	}
	return ids
}

func inputSourceLockAvailability(settings InputSourceLockSettings, availableSourceIDs map[string]bool) InputSourceLockAvailability {
	settings = normalizeInputSourceLockSettings(settings)
	availability := InputSourceLockAvailability{
		MissingSourceIDs: []string{},
		MissingAppRules:  []InputSourceLockMissingRule{},
	}
	if !settings.Enabled {
		return availability
	}
	if availableSourceIDs == nil {
		availability.RuntimeEnabled = inputSourceManagerConfig(settings).Enabled
		return availability
	}

	missingIDs := make(map[string]bool)
	addMissingID := func(sourceID string) {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" || missingIDs[sourceID] {
			return
		}
		missingIDs[sourceID] = true
		availability.MissingSourceIDs = append(availability.MissingSourceIDs, sourceID)
	}

	if settings.DefaultSourceID != "" && !availableSourceIDs[settings.DefaultSourceID] {
		availability.MissingDefaultSourceID = settings.DefaultSourceID
		addMissingID(settings.DefaultSourceID)
	}
	for _, rule := range settings.AppRules {
		if !rule.Enabled || rule.SourceID == "" || availableSourceIDs[rule.SourceID] {
			continue
		}
		availability.MissingAppRules = append(availability.MissingAppRules, InputSourceLockMissingRule{
			AppID:    rule.AppID,
			AppName:  rule.AppName,
			SourceID: rule.SourceID,
		})
		addMissingID(rule.SourceID)
	}
	sort.Strings(availability.MissingSourceIDs)
	availability.HasMissingSources = len(availability.MissingSourceIDs) > 0
	availability.RuntimeEnabled = inputSourceManagerConfigForAvailable(settings, availableSourceIDs).Enabled
	return availability
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
