package smtpin

import (
	"fmt"
)

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func printTraceLog(format string, a ...any) {
	if PrintTraceLogs {
		fmt.Printf(format, a...)
	}
}
