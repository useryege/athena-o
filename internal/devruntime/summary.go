package devruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AccessSettingsRead is a read-only observation, independent of process health.
// Settings retain the API's representation and are never replaced with defaults.
type AccessSettingsRead struct {
	Settings json.RawMessage `json:"settings,omitempty"`
	Error    string
	At       time.Time
}

func readAccessSettings(ctx context.Context, address string, env map[string]string) AccessSettingsRead {
	result := AccessSettingsRead{At: time.Now()}
	if address == "" {
		result.Error = "API Server is not selected; access settings were not queried"
		return result
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	target := "http://" + address + normalizedBase(env["ATHENA_SERVER_ROOTPATH"]) + "api/v1/admin/module-access-settings"
	body, _, err := readHTTP(bounded, client, target, "admin", false)
	if err != nil {
		result.Error = "access settings unavailable (administrator login may be required): " + err.Error()
		return result
	}
	var response struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err = json.Unmarshal(body, &response); err != nil {
		result.Error = "invalid access settings response: " + err.Error()
		return result
	}
	var settings []struct {
		ModuleKey string          `json:"module_key"`
		State     json.RawMessage `json:"state"`
	}
	if err = json.Unmarshal(response.Settings, &settings); err != nil || len(settings) == 0 {
		result.Error = "access settings response has no valid settings"
		return result
	}
	for _, setting := range settings {
		if setting.ModuleKey == "" || len(setting.State) == 0 {
			result.Error = "access settings response has an incomplete setting"
			return result
		}
	}
	result.Settings = response.Settings
	return result
}
func printRunSummary(state State, env map[string]string) {
	fmt.Printf("Instance %s: phase=%s, core usable=%t, selected ready=%t, full stack ready=%t\n", state.Key.Name, state.Phase, state.CoreUsable, state.SelectedReady, state.FullStackReady)
	for _, gateway := range state.Gateways {
		fmt.Printf("Gateway %s: serving=%t %s\n", gateway.Address, gateway.Serving, gateway.Error)
	}
	if state.GatewayError != "" {
		fmt.Printf("Gateway inspection: %s\n", state.GatewayError)
	}
	if state.AccessSettings.Error != "" {
		fmt.Println(state.AccessSettings.Error)
	} else {
		fmt.Printf("Access settings: %s\n", state.AccessSettings.Settings)
	}
	if address := state.Endpoints["ui"]; address != "" {
		base := normalizedBase(env["ATHENA_SERVER_BASEHREF"])
		fmt.Printf("Member: http://%s%s\nAdmin: http://%s%sadmin/\n", address, base, address, base)
	}
	fmt.Printf("Logs and state: %s\nStop: cd %s && make stop-instance INSTANCE=%s\nToken integration is deferred; readiness does not represent business acceptance.\n", state.Key.Dir(), shellQuote(state.Key.Checkout), shellQuote(state.Key.Name))
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func gatewayEnvironment(specs []ServiceSpec, env map[string]string) map[string]string {
	if selected(specs, "api-server") || selected(specs, "etherscan-manager") {
		return env
	}
	return nil
}
