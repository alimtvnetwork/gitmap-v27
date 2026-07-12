package cmd

import (
	"os"
	"strings"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
)

func applyCloneAssumeYesEnv(isAssumeYes bool) {
	if !isAssumeYes {
		return
	}
	cmd := withSSHAcceptNew(os.Getenv(constants.EnvGitSSHCommand))
	if err := os.Setenv(constants.EnvGitSSHCommand, cmd); err != nil {
		fmtCloneEnvError(err)
	}
}

func cloneEnvWithSSHAcceptNew() []string {
	cmd := withSSHAcceptNew(os.Getenv(constants.EnvGitSSHCommand))
	entry := constants.EnvGitSSHCommand + constants.EnvAssignmentSeparator + cmd
	return append(os.Environ(), entry)
}

func withSSHAcceptNew(existing string) string {
	trimmed := strings.TrimSpace(existing)
	if trimmed == "" {
		return constants.SSHBin + " " + constants.SSHOptionFlag + " " +
			constants.SSHStrictHostKeyAcceptNew
	}
	if strings.Contains(trimmed, constants.SSHStrictHostKeyChecking) {
		return trimmed
	}

	return trimmed + " " + constants.SSHOptionFlag + " " +
		constants.SSHStrictHostKeyAcceptNew
}

func isSSHCloneURL(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	return strings.HasPrefix(lower, constants.PrefixSSH) ||
		strings.HasPrefix(lower, constants.PrefixSSHScheme)
}
