package command

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Output_Succeeds_LeavingOutWarnings(t *testing.T) {
	cmd := exec.Command("sh", "-c", `echo 'WARNING: 192.168.1.161: server version 1.14.1 is older than client version 1.14.2' >&2; echo 'id: sda'`)

	output, err := Output(cmd)

	assert.NoError(t, err)
	assert.Equal(t, "id: sda\n", string(output))
}

func Test_Output_Fails_WithStdoutAndStderr(t *testing.T) {
	cmd := exec.Command("sh", "-c", `echo 'partial'; echo 'Error from server (NotFound)' >&2; exit 1`)

	output, err := Output(cmd)

	assert.EqualError(t, err, "exit status 1")
	assert.Equal(t, "partial\nError from server (NotFound)\n", string(output))
}

func Test_Output_Fails_WhenTheCommandIsMissing(t *testing.T) {
	output, err := Output(exec.Command("bbe-missing-command"))

	assert.True(t, errors.Is(err, exec.ErrNotFound))
	assert.Empty(t, output)
}
