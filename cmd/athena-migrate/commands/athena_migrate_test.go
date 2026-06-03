package commands

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/useryege/athena/internal/migration"
)

func TestUpAllModules(t *testing.T) {
	restore := replaceMigrationHooks(t, []migration.Module{{Name: "application"}, {Name: "wallet"}}, nil, nil)
	defer restore()

	var ran []string
	runUp = func(_ context.Context, module migration.Module) error {
		ran = append(ran, module.Name)
		return nil
	}

	cmd := NewCommand()
	cmd.SetArgs([]string{"up", "--module", "all"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if want := []string{"application", "wallet"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("run modules = %v, want %v", ran, want)
	}
}

func TestUpSingleModule(t *testing.T) {
	restore := replaceMigrationHooks(t, []migration.Module{{Name: "application"}, {Name: "wallet"}}, nil, nil)
	defer restore()

	var ran []string
	runUp = func(_ context.Context, module migration.Module) error {
		ran = append(ran, module.Name)
		return nil
	}

	cmd := NewCommand()
	cmd.SetArgs([]string{"up", "--module", "wallet"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if want := []string{"wallet"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("run modules = %v, want %v", ran, want)
	}
}

func TestUnknownModuleFails(t *testing.T) {
	restore := replaceMigrationHooks(t, []migration.Module{{Name: "application"}}, nil, nil)
	defer restore()

	cmd := NewCommand()
	cmd.SetArgs([]string{"up", "--module", "missing"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unknown module "missing"`) {
		t.Fatalf("Execute() error = %v, want unknown module", err)
	}
}

func TestStatusAllModules(t *testing.T) {
	restore := replaceMigrationHooks(t, []migration.Module{{Name: "application"}, {Name: "wallet"}}, nil, nil)
	defer restore()

	var ran []string
	runStatus = func(_ context.Context, module migration.Module) error {
		ran = append(ran, module.Name)
		return nil
	}

	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"status", "--module", "all"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if want := []string{"application", "wallet"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("status modules = %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), "application postgres migration status") {
		t.Fatalf("status output = %q, want application header", out.String())
	}
}

func TestRunnerErrorIncludesModule(t *testing.T) {
	restore := replaceMigrationHooks(t, []migration.Module{{Name: "wallet"}}, nil, errors.New("dsn missing"))
	defer restore()

	cmd := NewCommand()
	cmd.SetArgs([]string{"up", "--module", "wallet"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "wallet postgres migration up") {
		t.Fatalf("Execute() error = %v, want module context", err)
	}
}

func replaceMigrationHooks(t *testing.T, modules []migration.Module, statusErr error, upErr error) func() {
	t.Helper()
	previousSelect := selectModules
	previousUp := runUp
	previousStatus := runStatus
	selectModules = func(module string) ([]migration.Module, error) {
		if module == migration.AllModules {
			return modules, nil
		}
		for _, item := range modules {
			if item.Name == module {
				return []migration.Module{item}, nil
			}
		}
		return nil, errors.New(`unknown module "` + module + `"`)
	}
	runUp = func(context.Context, migration.Module) error {
		return upErr
	}
	runStatus = func(context.Context, migration.Module) error {
		return statusErr
	}
	return func() {
		selectModules = previousSelect
		runUp = previousUp
		runStatus = previousStatus
	}
}
