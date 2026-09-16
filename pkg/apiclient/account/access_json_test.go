package account

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestAccountAccessJSONIndependentFlags(t *testing.T) {
	modules := []AccountDataModule{1, 4, 8, 9, 11, 12, 13}
	var moduleJSON []string
	var grants []*AccountModuleAccess
	for _, module := range modules {
		moduleJSON = append(moduleJSON, fmt.Sprintf(`{"module":"%s","dataAccess":"ACCOUNT_DATA_ACCESS_READ"}`, module.String()))
		grants = append(grants, &AccountModuleAccess{Module: module, DataAccess: AccountDataAccess_ACCOUNT_DATA_ACCESS_READ})
	}
	for _, flags := range [][2]bool{{true, false}, {false, true}, {true, true}, {false, false}} {
		t.Run(fmt.Sprint(flags), func(t *testing.T) {
			source := fmt.Sprintf(`{"loginEnabled":true,"apiKeyEnabled":%t,"profitSharingEnabled":%t,"revision":"18446744073709551615","moduleAccess":[%s]}`, flags[0], flags[1], strings.Join(moduleJSON, ","))
			var decoded AccountAccess
			if err := json.Unmarshal([]byte(source), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.ApiKeyEnabled != flags[0] || decoded.ProfitSharingEnabled != flags[1] {
				t.Errorf("decoded independent flags: %t %t", decoded.ApiKeyEnabled, decoded.ProfitSharingEnabled)
			}
			original := AccountAccess{LoginEnabled: true, ApiKeyEnabled: flags[0], ProfitSharingEnabled: flags[1], Revision: ^uint64(0), ModuleAccess: grants}
			data, err := json.Marshal(&original)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if enabled, _ := wire["apiKeyEnabled"].(bool); enabled != flags[0] {
				t.Errorf("serialized apiKeyEnabled: %s", data)
			}
			if enabled, _ := wire["profitSharingEnabled"].(bool); enabled != flags[1] {
				t.Errorf("serialized profitSharingEnabled: %s", data)
			}
			if wire["revision"] != "18446744073709551615" || !decoded.LoginEnabled || decoded.Revision != ^uint64(0) || len(decoded.ModuleAccess) != len(modules) {
				t.Fatalf("other access facts lost: %s / %v", data, &decoded)
			}
			var roundtrip AccountAccess
			if err := json.Unmarshal(data, &roundtrip); err != nil {
				t.Fatal(err)
			}
			if len(roundtrip.ModuleAccess) != len(modules) {
				t.Fatalf("module matrix changed: %s", data)
			}
			for i, module := range modules {
				for _, access := range []*AccountModuleAccess{decoded.ModuleAccess[i], roundtrip.ModuleAccess[i]} {
					if access.Module != module || access.DataAccess != AccountDataAccess_ACCOUNT_DATA_ACCESS_READ {
						t.Fatalf("module access changed: %s", data)
					}
				}
			}
		})
	}
}
