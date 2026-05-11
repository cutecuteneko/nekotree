package testutil

import (
	"fmt"
	"testing"
)

// --- SequentialMockRunner ---

func TestSequentialMockRunner_ReturnsOutputsInOrder(t *testing.T) {
	seq := &SequentialMockRunner{
		Outputs: [][]byte{[]byte("first"), []byte("second")},
		Errs:    []error{nil, nil},
	}

	out1, err1 := seq.CombinedOutput("git", "status")
	if string(out1) != "first" || err1 != nil {
		t.Errorf("expected (first, nil), got (%s, %v)", string(out1), err1)
	}

	out2, err2 := seq.CombinedOutput("git", "log")
	if string(out2) != "second" || err2 != nil {
		t.Errorf("expected (second, nil), got (%s, %v)", string(out2), err2)
	}
}

func TestSequentialMockRunner_RepeatsLastEntryBeyondList(t *testing.T) {
	seq := &SequentialMockRunner{
		Outputs: [][]byte{[]byte("only")},
		Errs:    []error{nil},
	}

	_, _ = seq.CombinedOutput("cmd1")
	out, err := seq.CombinedOutput("cmd2") // beyond the list
	if string(out) != "only" || err != nil {
		t.Errorf("expected last entry repeated, got (%s, %v)", string(out), err)
	}
}

func TestSequentialMockRunner_RunRecordsCallAndReturnsErr(t *testing.T) {
	sentinel := fmt.Errorf("runner error")
	seq := &SequentialMockRunner{
		Outputs: [][]byte{nil},
		Errs:    []error{sentinel},
	}

	err := seq.Run("docker", "stop", "mycontainer")
	if err != sentinel {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if !seq.HasCall("docker stop mycontainer") {
		t.Errorf("expected call recorded, got: %v", seq.Calls)
	}
}

func TestSequentialMockRunner_HasCall(t *testing.T) {
	seq := &SequentialMockRunner{
		Outputs: [][]byte{nil},
		Errs:    []error{nil},
	}

	_, _ = seq.CombinedOutput("docker", "run", "alpine")

	if !seq.HasCall("docker run alpine") {
		t.Errorf("expected HasCall to find the call, got: %v", seq.Calls)
	}
	if seq.HasCall("docker stop") {
		t.Error("expected HasCall to return false for non-existent call")
	}
}

func TestSequentialMockRunner_ErrorsAndOutputsIndependent(t *testing.T) {
	seq := &SequentialMockRunner{
		Outputs: [][]byte{[]byte("output"), []byte("output2")},
		Errs:    []error{fmt.Errorf("err1")},
	}

	out1, err1 := seq.CombinedOutput("cmd1")
	if string(out1) != "output" || err1 == nil {
		t.Errorf("first call: expected (output, err), got (%s, %v)", out1, err1)
	}

	// Second call: Outputs[1] = "output2"; Errs exhausted → last entry (err1) repeated.
	out2, err2 := seq.CombinedOutput("cmd2")
	if string(out2) != "output2" || err2 == nil {
		t.Errorf("second call: expected (output2, err1 repeated), got (%s, %v)", out2, err2)
	}
}
