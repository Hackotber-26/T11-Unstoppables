package profiles

import (
	"errors"
	"strings"
)

type Preferences struct {
	MaxWalkMeters       int
	MaxWaitSeconds      int
	MaxDetourPercent    int
	RequiresAccessible  bool
	SafetyOptIn          bool
}

func (p Preferences) Validate() error {
	if p.MaxWalkMeters < 0 || p.MaxWalkMeters > 5000 {
		return errors.New("max walk distance must be between 0 and 5000 meters")
	}
	if p.MaxWaitSeconds < 0 || p.MaxWaitSeconds > 3600 {
		return errors.New("max wait must be between 0 and 3600 seconds")
	}
	if p.MaxDetourPercent < 0 || p.MaxDetourPercent > 100 {
		return errors.New("max detour must be between 0 and 100 percent")
	}
	return nil
}

func NormalizeDisplayName(value string) (string, error) {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) < 1 || len([]rune(value)) > 100 {
		return "", errors.New("display name must contain 1 to 100 characters")
	}
	return value, nil
}