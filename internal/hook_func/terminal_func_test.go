package hook_func

import (
	"context"
	"testing"
)

func TestTerminalHooksCanBeReusedAcrossSessions(t *testing.T) {
	called := 0

	BeginTerminalSession()
	RegisterTerminalFunc("first", func(context.Context) error {
		called++
		return nil
	})
	ExecTerminalFunc(context.Background())
	ExecTerminalFunc(context.Background())

	BeginTerminalSession()
	RegisterTerminalFunc("second", func(context.Context) error {
		called++
		return nil
	})
	ExecTerminalFunc(context.Background())

	if called != 2 {
		t.Fatalf("cleanup calls = %d, want 2", called)
	}
}
