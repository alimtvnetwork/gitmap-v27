// Package cmd — visibilityredo.go: `gitmap visibility-redo` (`vr`)
// reverses the most recent `VisibilityUndo` run, restoring the
// visibility state that the undo reverted. Pure reuse of the
// shared reverseRunAndExit helper from visibilityundo.go — the only
// differences are the source-run filter (CommandKind='VisibilityUndo')
// and the cmdName under which the new audit run is logged.
//
// Accepts `--run <id>` to redo a specific historical undo run.
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §undo-redo.
package cmd

import "github.com/alimtvnetwork/gitmap-v25/gitmap/constants"

// runVisibilityRedo is the dispatcher entry point.
func runVisibilityRedo(args []string) {
	flags := parseUndoArgs(args)
	run, results := loadReversible(flags.RunID, constants.CommandKindVisibilityUndo, constants.ErrRedoNoRunFound)
	reverseRunAndExit(run, results, flags, constants.CmdVisibilityRedo)
}
