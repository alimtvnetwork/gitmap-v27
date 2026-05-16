package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v19/gitmap/completion"
	"github.com/alimtvnetwork/gitmap-v19/gitmap/constants"
)

// verifyShellWrapper checks if the shell wrapper is active after setup.
func verifyShellWrapper(dryRun bool) {
	if dryRun {
		return
	}

	shell := completion.DetectShell()
	fmt.Printf("\n  %s■ Shell Wrapper Verify —%s\n", constants.ColorYellow, constants.ColorReset)

	if isWrapperActive() {
		fmt.Printf(constants.MsgWrapperVerifyOK, constants.ColorGreen, constants.ColorReset)

		return
	}

	printWrapperReloadTip(shell)
}

// isWrapperActive returns true if the command wrapper env var is set.
func isWrapperActive() bool {
	return os.Getenv(constants.EnvGitmapCommandWrapper) == constants.EnvGitmapWrapperVal
}

// printWrapperReloadTip prints reload instructions for the detected shell.
func printWrapperReloadTip(shell string) {
	fmt.Printf(constants.MsgWrapperVerifyTip,
		constants.ColorYellow, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
	)
}

// warnIfNoWrapper prints a stderr warning when cd is called without wrapper.
func warnIfNoWrapper() {
	if isWrapperActive() {
		return
	}

	installWrapperForCurrentShell()
	printNoWrapperWarning()
}

func installWrapperForCurrentShell() {
	shell := completion.DetectShell()
	if err := completion.InstallCDFunction(shell); err != nil {
		fmt.Fprintf(os.Stderr, constants.WarnWrapperInstallFmt, err)
	}
}

func printNoWrapperWarning() {
	fmt.Fprintf(os.Stderr, constants.MsgWrapperNotLoaded,
		constants.ColorYellow, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
		constants.ColorCyan, constants.ColorReset,
	)
}
