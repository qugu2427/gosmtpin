package smtpin

import (
	"regexp"
	"strings"
)

var emailRgx *regexp.Regexp = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)

/*
	find an <email> in a string between '<' and  '>'

	example: "args: args <bob@gmail.com> ... arg4" -> "bob@gmail.com"
*/
func findEmailInLine(line string) (found bool, email string) {
	lessThanIndex := -1
	greaterThanIndex := -1

	for i, char := range line {
		if char == '>' {
			greaterThanIndex = i
		} else if char == '<' && lessThanIndex == -1 {
			lessThanIndex = i
		}
	}

	if lessThanIndex == -1 ||
		greaterThanIndex == -1 ||
		lessThanIndex >= greaterThanIndex {
		return false, ""
	} else {
		email = line[lessThanIndex+1 : greaterThanIndex]
		found = emailRgx.MatchString(email)
		return
	}

}

/*
parse string to args by ignoring ':' and ' '

example: "arg1: arg2 arg3:arg4" -> [arg1, arg2, arg3, arg4]
*/
func argSplit(str string) (args []string) {
	curr := ""
	for _, c := range str {
		if c == ' ' || c == ':' {
			if curr != "" {
				args = append(args, curr)
				curr = ""
			}
		} else {
			curr += string(c)
		}
	}
	if curr != "" {
		args = append(args, curr)
	}
	return
}

/*
clips body to trailing period.

example: "bla bla bla\r\n.\r\n\r\n" -> "bla bla bla\r\n."
*/
func clipBody(body string) (clippedBody string) {
	clipIndex := strings.LastIndex(body, bodyEnd)
	clippedBody = body[:clipIndex+3]
	return clippedBody
}
