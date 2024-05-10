package context

import (
	"os"
)

const ERROR_STATUS_CODE_FAILURE = -6
const ERROR_NO_RESPONSE = -7
const ERROR_INVALID_URL = -8
const ERROR_CANNOT_READ_FILE = -9
const ERROR_CANNOT_WRITE_FILE = -10
const ERROR_CANNOT_WRITE_TO_STDOUT = -11

func (ctx *CurlContext) HandleErrorAndExit(err error, exitCode int, entry string) {
	if err == nil {
		return
	}
	if entry == "" {
		entry = "Error"
	}
	entry += ": "
	entry += err.Error()

	if exitCode == ERROR_CANNOT_WRITE_TO_STDOUT {
		// don't recurse (it called us to report the failure to write errors to a normal file)
		panic(err)
	}

	if (!ctx.IsSilent && !ctx.SilentFail) || !ctx.ShowErrorEvenIfSilent {
		writeToFileBytes(ctx.ErrorOutput, []byte(entry))
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
