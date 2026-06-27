package taskx

import "github.com/caiflower/dagflow/taskx/types"

func isFinished(state string) bool {
	return state == string(types.TaskFailed) || state == string(types.TaskSucceeded)
}

func isRollbackFinished(state string) bool {
	return state == string(types.RollbackFailed) || state == string(types.RollbackSucceeded) || state == string(types.NoneRollback)
}
