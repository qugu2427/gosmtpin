package smtpin

import "testing"

func TestIsValidHelo(t *testing.T) {
	validHelos := []string{
		".",
		"HE_lo-test.com",
	}

	for _, helo := range validHelos {
		if !isValidHelo(helo) {
			t.Fatalf("isValidHelo(\"%s\") = true, but got false", helo)
		}
	}
}
