package smtpin

import "testing"

func TestHandleHelo(t *testing.T) {
	type testCase struct {
		domain        string
		beforeSession session
		afterSession  session
		response      response
	}

	testCases := []testCase{
		{
			"localhost",
			createNewSession(),
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resOk,
		},
		{
			"127.0.0.1:8000",
			createNewSession(),
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "127.0.0.1:8000",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resOk,
		},
		{
			"#!$@#$@# fsdf",
			createNewSession(),
			createNewSession(),
			resSyntaxError,
		},
		{
			"",
			createNewSession(),
			createNewSession(),
			resSyntaxError,
		},
		{
			"localhost",
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resInvalidSequence,
		},
	}

	for _, testCase := range testCases {
		res := testCase.beforeSession.handleHelo(testCase.domain)
		if !res.equals(testCase.response) {
			t.Fatalf("session.handleHelo(%#v) = %#v expected %#v", testCase.domain, res, testCase.response)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleHelo(%#v) resulted in session %#v expected %#v", testCase.domain, testCase.beforeSession, testCase.afterSession)
		}
	}
}
