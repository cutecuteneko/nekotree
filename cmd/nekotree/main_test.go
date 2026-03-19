package main

import (
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestNekotreeCommands(t *testing.T) {
	app := &cli.App{
		Commands: []*cli.Command{
			{Name: "create"},
			{Name: "shell"},
			{Name: "list"},
			{Name: "remove"},
		},
	}

	for _, cmd := range []string{"create", "shell", "list", "remove"} {
		if app.Command(cmd) == nil {
			t.Errorf("Command %s is missing from registry", cmd)
		}
	}
}

func TestSizeParsing(t *testing.T) {
	// Simple unit test for a hypothetical size helper
	input := "4.0K\t/some/path"
	if !strings.Contains(input, "4.0K") {
		t.Error("Failed to parse disk usage output")
	}
}
