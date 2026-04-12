package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentLogsCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent logs: not implemented yet")
	return nil
}
