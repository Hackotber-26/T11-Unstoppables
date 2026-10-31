package users

import "testing"

func TestRolePermissionsAreLeastPrivilege(t *testing.T) {
	if !Can(Rider, "trip:create") || Can(Rider, "wallet:credit") || !Can(Admin, "wallet:credit") {
		t.Fatal("unexpected role permission matrix")
	}
}