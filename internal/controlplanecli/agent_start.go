package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentStartCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent start: not implemented yet")
	return nil
}
