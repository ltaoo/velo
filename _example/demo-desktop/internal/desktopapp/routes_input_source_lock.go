package desktopapp

import (
	"encoding/json"
	"errors"
	"runtime"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/inputsource"
)

func registerInputSourceLockRoutes(b *velo.Box, service *InputSourceLockService) {
	b.Get("/api/settings/input-source-lock", func(c *velo.BoxContext) interface{} {
		raw := b.Store.Get(inputSourceLockSettingsKey)
		settings, err := loadStoredInputSourceLockSettings(raw)
		if err != nil {
			return c.Error(err.Error())
		}
		status := inputSourceLockStatus(settings)
		return c.Ok(velo.H{
			"found":   raw != nil,
			"config":  settings,
			"status":  status,
			"runtime": velo.H{"os": runtime.GOOS},
		})
	})

	b.Post("/api/settings/input-source-lock/save", func(c *velo.BoxContext) interface{} {
		var settings InputSourceLockSettings
		if err := c.BindJSON(&settings); err != nil {
			return c.Error(err.Error())
		}
		settings = normalizeInputSourceLockSettings(settings)
		raw, err := marshalInputSourceLockSettingsForStore(settings)
		if err != nil {
			return c.Error(err.Error())
		}
		if err := b.Store.Set(inputSourceLockSettingsKey, json.RawMessage(raw)); err != nil {
			return c.Error(err.Error())
		}
		if service != nil {
			service.Apply(settings)
		}
		return c.Ok(velo.H{"success": true, "config": settings, "status": inputSourceLockStatus(settings)})
	})

	b.Get("/api/input-source/status", func(c *velo.BoxContext) interface{} {
		settings, _ := loadStoredInputSourceLockSettings(b.Store.Get(inputSourceLockSettingsKey))
		return c.Ok(inputSourceLockStatus(settings))
	})
}

func inputSourceLockStatus(settings InputSourceLockSettings) velo.H {
	sources, sourceErr := inputsource.List()
	current, currentErr := inputsource.Current()
	frontmost, frontmostErr := inputsource.FrontmostApp()
	supported := sourceErr == nil
	if errors.Is(sourceErr, inputsource.ErrUnsupported) || errors.Is(currentErr, inputsource.ErrUnsupported) {
		supported = false
	}
	var availability InputSourceLockAvailability
	if sourceErr == nil {
		availability = inputSourceLockAvailability(settings, inputSourceIDSet(sources))
	} else {
		availability = InputSourceLockAvailability{
			MissingSourceIDs: []string{},
			MissingAppRules:  []InputSourceLockMissingRule{},
		}
	}
	return velo.H{
		"enabled":                normalizeInputSourceLockSettings(settings).Enabled,
		"runtimeEnabled":         availability.RuntimeEnabled,
		"supported":              supported,
		"sources":                sources,
		"current":                current,
		"frontmostApp":           frontmost,
		"sourceError":            errorString(sourceErr),
		"currentError":           errorString(currentErr),
		"frontmostError":         errorString(frontmostErr),
		"hasMissingSources":      availability.HasMissingSources,
		"missingDefaultSourceId": availability.MissingDefaultSourceID,
		"missingSourceIds":       availability.MissingSourceIDs,
		"missingAppRules":        availability.MissingAppRules,
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
