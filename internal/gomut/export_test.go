package gomut

import "io"

var (
	PrepareRunRootWithCopyExclude = prepareRunRoot
	GoCommandEnv                  = goCommandEnv
	NewRootCommand                = (*Command).newRootCommand
	BuildTestRunConfig            = (*Command).buildTestRunConfig
	RunCandidateLoop              = (*Runner).runCandidateLoop
)

var NewRunnerWithExecuteMutation = func(stdout, stderr io.Writer, executeMutationFunc ExecuteMutationFunc) *Runner {
	runner := NewRunner(stdout, stderr)
	runner.executeMutationFunc = executeMutationFunc

	return runner
}
