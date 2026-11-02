package riders

import (
	"errors"
	"net/mail"
	"strings"
)

type Profile struct {
	ID        string
	Name      string
	Email     string
	Phone     string
	StationID string
}

func (profile Profile) Validate() error {
	if profile.ID == "" || strings.TrimSpace(profile.Name) == "" || len(profile.Name) > 100 {
		return errors.New("rider id and display name are required")
	}
	parsed, err := mail.ParseAddress(profile.Email)
	if err != nil || parsed.Address != profile.Email || len(profile.Email) > 254 {
		return errors.New("rider email is invalid")
	}
	if len(profile.Phone) > 40 || len(profile.StationID) > 100 {
		return errors.New("rider profile field exceeds limits")
	}
	return nil
}