package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentStatusCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent status: not implemented yet")
	return nil
}
