package smtpin

import (
	"fmt"
	"regexp"
)

var (
	emailRgx = regexp.MustCompile(`^[^\s@]+@[a-zA-Z0-9-\./_:]+$`)
	heloRgx  = regexp.MustCompile(`^[a-zA-Z0-9-\./_:]+$`)
)

func extractAddress(bracketedAddress string) (address string, isValid bool) {
	lenBracketedAddress := len(bracketedAddress)
	if lenBracketedAddress == 0 || bracketedAddress[0] != '<' || bracketedAddress[lenBracketedAddress-1] != '>' {
		return "", false
	}
	address = bracketedAddress[1 : lenBracketedAddress-1]
	isValid = emailRgx.MatchString(address)
	if !isValid {
		address = ""
	}
	return
}

func isValidHelo(helo string) (isValid bool) {
	return heloRgx.MatchString(helo)
}

func printTraceLog(format string, a ...any) {
	if PrintTraceLogs {
		fmt.Printf(format, a...)
	}
}
