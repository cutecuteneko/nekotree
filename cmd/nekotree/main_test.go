package main

import (
	"testing"
	"github.com/urfave/cli/v2"
)

func TestCreateCommandFlags(t *testing.T) {
	cmd := createCommand()
	
	expectedFlags := []string{"branch", "name", "image"}
	for _, fName := range expectedFlags {
		found := false
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.StringFlag); ok && f.Name == fName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected flag %s not found in create command", fName)
		}
	}
}

func TestRemoveCommandForceFlag(t *testing.T) {
	cmd := removeCommand()
	found := false
	for _, flag := range cmd.Flags {
		if f, ok := flag.(*cli.BoolFlag); ok && f.Name == "force" {
			found = true
		}
	}
	if !found {
		t.Error("Remove command missing 'force' bool flag")
	}
}
