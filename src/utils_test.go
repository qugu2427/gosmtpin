package smtpin

import "testing"

func TestExtractAddress(t *testing.T) {
	type testCase struct {
		bracketedAddress string
		address          string
		isValid          bool
	}

	testCases := []testCase{
		{"<a@b.c>", "a@b.c", true},
		{"<complicated.name-_@127.0.0.1>", "complicated.name-_@127.0.0.1", true},
		{"<a@localhost>", "a@localhost", true},
		{"<>", "", false},
		{"", "", false},
		{"<invalid name@c.com>", "", false},
	}

	for _, testCase := range testCases {
		address, isValid := extractAddress(testCase.bracketedAddress)
		if address != testCase.address || isValid != testCase.isValid {
			t.Fatalf(
				"extractAddress(%#v) = %#v, %t expected %#v, %t",
				testCase.bracketedAddress,
				address, isValid,
				testCase.address, testCase.isValid,
			)
		}
	}
}

func TestIsValidHelo(t *testing.T) {
	type testCase struct {
		helo    string
		isValid bool
	}

	testCases := []testCase{
		{"localhost", true},
		{"127.0.0.1", true},
		{"invalid domain", false},
		{"", false},
	}

	for _, testCase := range testCases {
		isValid := isValidHelo(testCase.helo)
		if isValid != testCase.isValid {
			t.Fatalf(
				"isValidHelo(%#v) = %t expected %t",
				testCase.helo, isValid, testCase.isValid,
			)
		}
	}
}
