package shared

import (
	"fmt"
	"os"

	"github.com/lucasepe/x/cl"
)

// FailTask prints an error and maps it to a command failure exit status.
func FailTask(err error) cl.ExitStatus {
	fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	return cl.ExitFailure
}
