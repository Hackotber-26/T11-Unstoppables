package users

import "errors"

type Role string

const (
	Rider Role = "rider"
	Driver Role = "driver"
	Admin Role = "admin"
)

func Can(role Role, action string) bool {
	permissions := map[Role]map[string]bool{
		Rider: {"trip:create": true, "trip:read:own": true, "pool:read": true, "wallet:read:own": true, "wallet:pay:own": true},
		Driver: {"trip:read:assigned": true, "trip:update:assigned": true, "availability:update:own": true},
		Admin: {"rider:list": true, "driver:list": true, "metrics:read": true, "simulation:run": true, "wallet:credit": true, "incident:manage": true},
	}
	return permissions[role][action]
}

func ValidateRole(role Role) error {
	if role != Rider && role != Driver && role != Admin {
		return errors.New("unsupported user role")
	}
	return nil
}