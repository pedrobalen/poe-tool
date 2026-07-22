package app

import (
	"log"
	"strconv"
)

const (
	keyLocked  = "overlay_locked"
	keyOpacity = "overlay_opacity"
	keyCompare = "overlay_compare"

	defaultOpacity = 1.0
)

// preferences holds the persisted overlay settings.
type preferences struct {
	locked  bool
	opacity float64
	compare bool
}

// loadPreferences reads the persisted overlay settings, falling back to sensible
// defaults when unset or malformed.
func (a *App) loadPreferences() preferences {
	prefs := preferences{opacity: defaultOpacity, compare: true}

	if v, ok, err := a.deps.SettingsRepo.Get(a.ctx, keyLocked); err != nil {
		log.Printf("reading lock preference: %v", err)
	} else if ok {
		prefs.locked = v == "1"
	}

	if v, ok, err := a.deps.SettingsRepo.Get(a.ctx, keyOpacity); err != nil {
		log.Printf("reading opacity preference: %v", err)
	} else if ok {
		if parsed, perr := strconv.ParseFloat(v, 64); perr == nil {
			prefs.opacity = parsed
		}
	}

	if v, ok, err := a.deps.SettingsRepo.Get(a.ctx, keyCompare); err != nil {
		log.Printf("reading compare preference: %v", err)
	} else if ok {
		prefs.compare = v == "1"
	}

	return prefs
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
