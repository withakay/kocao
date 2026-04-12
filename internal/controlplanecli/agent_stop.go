package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentStopCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent stop: not implemented yet")
	return nil
}
