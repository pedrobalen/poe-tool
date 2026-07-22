package app

import (
	"log"
	"strconv"
)

const (
	keyLocked  = "overlay_locked"
	keyOpacity = "overlay_opacity"

	defaultOpacity = 1.0
)

// loadPreferences reads the persisted overlay lock and opacity, falling back to
// sensible defaults when unset or malformed.
func (a *App) loadPreferences() (locked bool, opacity float64) {
	opacity = defaultOpacity

	if v, ok, err := a.deps.SettingsRepo.Get(a.ctx, keyLocked); err != nil {
		log.Printf("reading lock preference: %v", err)
	} else if ok {
		locked = v == "1"
	}

	if v, ok, err := a.deps.SettingsRepo.Get(a.ctx, keyOpacity); err != nil {
		log.Printf("reading opacity preference: %v", err)
	} else if ok {
		if parsed, perr := strconv.ParseFloat(v, 64); perr == nil {
			opacity = parsed
		}
	}

	return locked, opacity
}

func (a *App) savePreference(key, value string) {
	if err := a.deps.SettingsRepo.Set(a.ctx, key, value); err != nil {
		log.Printf("saving preference %q: %v", key, err)
	}
}

func boolToSetting(b bool) string {
	if b {
		return "1"
	}

	return "0"
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
