package command

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
)

// Output runs the command and returns what it printed to stdout. Tools like talosctl print warnings to stderr
// even when they succeed, such as a client newer than the node, which would break parsing their output. When
// the command fails, stderr is returned as well, as that's where the reason is
func Output(cmd *exec.Cmd) ([]byte, error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.Output()
	if stderr.Len() > 0 {
		logger.Debug(fmt.Sprintf("Stderr: %s", stderr.String()))
	}

	if err != nil {
		return append(stdout, stderr.Bytes()...), err
	}

	return stdout, nil
}
