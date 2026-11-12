package roles_test

import (
	"slices"
	"testing"

	"github.com/gshpychka/fob/internal/roles"
)

func TestProfilesTellApartAccountsThatShareAName(t *testing.T) {
	profiles := roles.Profiles([]roles.Role{
		{AccountID: "333333333333", AccountName: "sandbox", Name: "Admin"},
		{AccountID: "111111111111", AccountName: "Data Platform", Name: "Admin"},
		{AccountID: "222222222222", AccountName: "sandbox", Name: "Admin"},
	})
	var names []string
	for _, profile := range profiles {
		names = append(names, profile.Name)
	}
	want := []string{"Data-Platform/Admin", "sandbox-222222222222/Admin", "sandbox-333333333333/Admin"}
	if !slices.Equal(names, want) {
		t.Errorf("profiles are named %v, want %v", names, want)
	}
}
