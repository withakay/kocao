package controlplanecli

import (
	"fmt"
	"io"
)

func runAgentListCommand(_ []string, _ Config, stdout io.Writer, _ io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "agent list: not implemented yet")
	return nil
}
