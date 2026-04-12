package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentExecCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent exec: not implemented yet")
	return nil
}
