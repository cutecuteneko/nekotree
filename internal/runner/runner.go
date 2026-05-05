package runner

import "os/exec"

// CommandRunner allows mocking shell execution in unit tests.
type CommandRunner interface {
	Run(name string, arg ...string) error
	CombinedOutput(name string, arg ...string) ([]byte, error)
}

// RealRunner is the production implementation using os/exec.
type RealRunner struct{}

func (r *RealRunner) Run(name string, arg ...string) error {
	// #nosec G204 - Variables are sanitized by calling packages using internal/utils
	return exec.Command(name, arg...).Run()
}

func (r *RealRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	// #nosec G204 - Variables are sanitized by calling packages using internal/utils
	return exec.Command(name, arg...).CombinedOutput()
}
