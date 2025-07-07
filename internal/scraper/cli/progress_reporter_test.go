package cli

import (
	"testing"
)

func TestProgressReporter_Basic(t *testing.T) {
	r := NewProgressReporter(10, true)
	r.UpdateProgress(5, 1, 2)
	r.UpdateWithMessage(6, 1, 2, "halfway done")
	r.LogError("something went wrong")
	r.LogInfo("info message")
	r.LogSuccess("success message")
	r.IncrementRetries()
	r.RecordError("timeout")
	r.Finish()
	breakdown := r.GetErrorBreakdown()
	if breakdown["timeout"] != 1 {
		t.Errorf("expected 1 timeout error, got %d", breakdown["timeout"])
	}
}
