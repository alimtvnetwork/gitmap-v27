package cmd

import (
	"fmt"
	"os"
)

func fmtCloneEnvError(err error) {
	fmt.Fprintf(os.Stderr, "clone: failed to configure SSH host-key acceptance: %v\n", err)
}
