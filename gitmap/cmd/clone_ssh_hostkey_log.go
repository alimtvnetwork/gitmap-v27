package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
)

func fmtCloneEnvError(err error) {
	fmt.Fprintf(os.Stderr, constants.ErrCloneSSHEnvFmt, err)
}
