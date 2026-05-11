// Package testutil provides shared test helpers for nekotree unit tests.
package testutil

import (
	"fmt"
	"strings"
)

// MockRunner records every runner call and returns configurable output/error.
// Use it anywhere a runner.CommandRunner is accepted to avoid real Docker/git
// execution in unit tests.
type MockRunner struct {
	Calls  []string
	Output []byte
	Err    error
}

func (m *MockRunner) Run(name string, arg ...string) error {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return m.Err
}

func (m *MockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return m.Output, m.Err
}

// HasCall reports whether any recorded call contains substr.
func (m *MockRunner) HasCall(substr string) bool {
	for _, c := range m.Calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// SequentialMockRunner returns outputs in sequence; the last entry is repeated
// for any calls beyond the provided list. Use this when a single action makes
// multiple runner calls that need different return values.
type SequentialMockRunner struct {
	Calls   []string
	Outputs [][]byte
	Errs    []error
	idx     int
}

func (m *SequentialMockRunner) next() ([]byte, error) {
	i := m.idx
	m.idx++

	var out []byte
	if i < len(m.Outputs) {
		out = m.Outputs[i]
	} else if len(m.Outputs) > 0 {
		out = m.Outputs[len(m.Outputs)-1]
	}

	var err error
	if i < len(m.Errs) {
		err = m.Errs[i]
	} else if len(m.Errs) > 0 {
		err = m.Errs[len(m.Errs)-1]
	}

	return out, err
}

func (m *SequentialMockRunner) Run(name string, arg ...string) error {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	_, err := m.next()
	return err
}

func (m *SequentialMockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return m.next()
}

// HasCall reports whether any recorded call contains substr.
func (m *SequentialMockRunner) HasCall(substr string) bool {
	for _, c := range m.Calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}
