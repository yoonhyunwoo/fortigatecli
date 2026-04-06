package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fortigatecli/internal/fortigate"
	"github.com/spf13/cobra"
)

func TestSystemReadAliases(t *testing.T) {
	got := map[string]readAlias{}
	for _, alias := range systemReadAliases {
		got[alias.use] = alias
	}

	tests := []struct {
		name string
		path string
		kind string
	}{
		{name: "admins", path: "system/admin", kind: "cmdb"},
		{name: "dns", path: "system/dns", kind: "cmdb"},
		{name: "ntp", path: "system/ntp", kind: "cmdb"},
		{name: "vdoms", path: "system/vdom", kind: "cmdb"},
	}
	for _, tc := range tests {
		alias, ok := got[tc.name]
		if !ok {
			t.Fatalf("missing alias %q", tc.name)
		}
		if alias.path != tc.path || alias.kind != tc.kind {
			t.Fatalf("%s mismatch: %#v", tc.name, alias)
		}
	}
}

func TestSystemMonitorCompatibilitySpecs(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "status", path: "system/status"},
		{name: "interfaces", path: "system/interface"},
		{name: "ha-status", path: "system/ha-status"},
		{name: "license", path: "license/status"},
	}
	got := map[string]monitorEndpointSpec{}
	for _, spec := range systemMonitorCompatibilitySpecs() {
		got[spec.use] = spec
	}
	for _, tc := range tests {
		spec, ok := got[tc.name]
		if !ok {
			t.Fatalf("missing compatibility spec %q", tc.name)
		}
		if spec.path != tc.path {
			t.Fatalf("%s path = %q", tc.name, spec.path)
		}
	}
}

func TestReadFlagsSupportAllVDOMs(t *testing.T) {
	root := newRootCommand()
	tests := [][]string{
		{"cmdb", "get"},
		{"cmdb", "list"},
		{"monitor", "get"},
		{"raw", "get"},
	}
	for _, path := range tests {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) error = %v", path, err)
		}
		if cmd.Flags().Lookup("all-vdoms") == nil {
			t.Fatalf("%v missing --all-vdoms flag", path)
		}
	}
}

type stubBackupRunner struct {
	plan *fortigate.BackupPlan
	data []byte
	err  error
}

func (s stubBackupRunner) BackupWithOptions(context.Context, fortigate.BackupOptions) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBackupRunner) BackupPlan(fortigate.BackupOptions) (*fortigate.BackupPlan, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.plan, nil
}

func TestSystemBackupCommandIncludesExportSubcommand(t *testing.T) {
	cmd := newSystemCommand(&rootOptions{})
	backupCmd, _, err := cmd.Find([]string{"backup"})
	if err != nil {
		t.Fatalf("Find(backup) error = %v", err)
	}
	exportCmd, _, err := cmd.Find([]string{"backup", "export"})
	if err != nil {
		t.Fatalf("Find(backup export) error = %v", err)
	}
	if !strings.Contains(backupCmd.Long, "stdout-only") {
		t.Fatalf("backup help = %q", backupCmd.Long)
	}
	if !strings.Contains(exportCmd.Long, "--output") {
		t.Fatalf("export help = %q", exportCmd.Long)
	}
}

func TestParseBackupScope(t *testing.T) {
	scope, err := parseBackupScope("VDOM")
	if err != nil {
		t.Fatalf("parseBackupScope() error = %v", err)
	}
	if scope != fortigate.BackupScopeVDOM {
		t.Fatalf("scope = %q", scope)
	}
	if _, err := parseBackupScope("invalid"); err == nil {
		t.Fatal("parseBackupScope() error = nil, want error")
	}
}

func TestBackupCommandOptionsRejectsGlobalScopeWithVDOMOverride(t *testing.T) {
	cmd := &cobra.Command{Use: "backup"}
	cmd.Flags().String("vdom", "", "override VDOM")
	if err := cmd.Flags().Set("vdom", "edge"); err != nil {
		t.Fatalf("Set(vdom) error = %v", err)
	}
	if _, err := (&backupCommandOptions{scope: "global"}).toAPIOptions(cmd, true); err == nil {
		t.Fatal("toAPIOptions() error = nil, want error")
	}
}

func TestRunBackupExportDryRunHasNoSideEffect(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "config.conf")
	cmd := &cobraTestCommand{buffer: new(bytes.Buffer)}

	err := runBackupExport(cmd.command(), stubBackupRunner{
		plan: &fortigate.BackupPlan{
			URL:   "https://fg/api/v2/monitor/system/config/backup?scope=global",
			Scope: fortigate.BackupScopeGlobal,
		},
	}, "json", fortigate.BackupOptions{
		Scope:      fortigate.BackupScopeGlobal,
		OutputPath: outputPath,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("runBackupExport() error = %v", err)
	}
	if _, statErr := os.Stat(outputPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output file stat = %v, want not exist", statErr)
	}
	if !strings.Contains(cmd.buffer.String(), "\"output\":") {
		t.Fatalf("dry-run output = %q", cmd.buffer.String())
	}
}

func TestRunBackupExportFailsWhenOutputExistsWithoutForce(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "config.conf")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cmd := &cobraTestCommand{buffer: new(bytes.Buffer)}

	err := runBackupExport(cmd.command(), stubBackupRunner{
		plan: &fortigate.BackupPlan{URL: "https://fg/api/v2/monitor/system/config/backup?scope=global", Scope: fortigate.BackupScopeGlobal},
		data: []byte("new-data"),
	}, "json", fortigate.BackupOptions{Scope: fortigate.BackupScopeGlobal, OutputPath: outputPath})
	if err == nil {
		t.Fatal("runBackupExport() error = nil, want error")
	}
	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(data) != "existing" {
		t.Fatalf("file contents = %q", string(data))
	}
}

func TestRunBackupExportAllowsForceOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "config.conf")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cmd := &cobraTestCommand{buffer: new(bytes.Buffer)}

	err := runBackupExport(cmd.command(), stubBackupRunner{
		plan: &fortigate.BackupPlan{URL: "https://fg/api/v2/monitor/system/config/backup?scope=vdom&vdom=edge", Scope: fortigate.BackupScopeVDOM, VDOM: "edge"},
		data: []byte("new-data"),
	}, "json", fortigate.BackupOptions{
		Scope:      fortigate.BackupScopeVDOM,
		VDOM:       "edge",
		OutputPath: outputPath,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("runBackupExport() error = %v", err)
	}
	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(data) != "new-data" {
		t.Fatalf("file contents = %q", string(data))
	}
}

type cobraTestCommand struct {
	buffer *bytes.Buffer
}

func (c *cobraTestCommand) command() *cobra.Command {
	cmd := newRootCommand()
	cmd.SetOut(c.buffer)
	cmd.SetErr(c.buffer)
	return cmd
}
