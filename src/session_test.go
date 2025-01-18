package smtpin

import (
	"fmt"
	"net"
	"testing"
)

func TestHandleHelo(t *testing.T) {
	type testCase struct {
		domain        string
		beforeSession session
		afterSession  session
		response      response
	}

	testCases := []testCase{

		// ok helos
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

		// invalid helos
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

		// invalid sequence
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

func TestHandleEhlo(t *testing.T) {
	type testCase struct {
		domain        string
		beforeSession session
		afterSession  session
		response      response
	}

	maxMsgSize := 1234

	testCases := []testCase{
		// ok ehlos
		{
			"localhost",
			createNewSession(),
			session{
				flags:      sessionFlagSaidHello | sessionFlagExtended,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			response{
				statusCode:   codeOk,
				msg:          resOk.msg,
				extendedMsgs: []string{"PIPELINING", "STARTTLS", fmt.Sprintf("SIZE %d", maxMsgSize)},
			},
		},
		{
			"127.0.0.1:8000",
			createNewSession(),
			session{
				flags:      sessionFlagSaidHello | sessionFlagExtended,
				helloFrom:  "127.0.0.1:8000",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			response{
				statusCode:   codeOk,
				msg:          resOk.msg,
				extendedMsgs: []string{"PIPELINING", "STARTTLS", fmt.Sprintf("SIZE %d", maxMsgSize)},
			},
		},

		// invalid ehlos
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

		// invalid sequence
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
		res := testCase.beforeSession.handleEhlo(testCase.domain, maxMsgSize)
		if !res.equals(testCase.response) {
			t.Fatalf("session.handleEhlo(%#v, %#v) = %#v expected %#v", testCase.domain, maxMsgSize, res, testCase.response)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleEhlo(%#v, %#v) resulted in session %#v expected %#v", testCase.domain, maxMsgSize, testCase.beforeSession, testCase.afterSession)
		}
	}
}

func TestHandleMailFrom(t *testing.T) {
	type testCase struct {
		senderIp      net.IP
		handleSpf     SpfHandler
		tlsMode       ListenerTlsMode
		address       string
		beforeSession session
		afterSession  session
		response      response
	}

	testCases := []testCase{

		// ok exchange
		{
			nil,
			nil,
			TlsModeNone,
			"<bob@localhost>",
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "bob@localhost",
				recipients: []string{},
				body:       "",
			},
			resOk,
		},

		// invalid sequence
		{
			nil,
			nil,
			TlsModeNone,
			"<bob@localhost>",
			session{
				flags:      0,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      0,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resInvalidSequence,
		},

		// invalid address
		{
			nil,
			nil,
			TlsModeNone,
			"bob@localhost>",
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resSyntaxError,
		},

		// tls required
		{
			nil,
			nil,
			TlsModeStartTls,
			"<bob@localhost>",
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      sessionFlagSaidHello,
				helloFrom:  "localhost",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resTlsRequired,
		},
	}

	for _, testCase := range testCases {
		res := testCase.beforeSession.handleMailFrom(
			testCase.senderIp,
			testCase.handleSpf,
			testCase.tlsMode,
			testCase.address,
		)
		if !res.equals(testCase.response) {
			t.Fatalf("session.handleMailFrom(%#v, func, %#v, %#v) = %#v expected %#v",
				testCase.senderIp,
				testCase.tlsMode,
				testCase.address,
				res,
				testCase.response,
			)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleMailFrom(%#v, func, %#v, %#v) resulted in session %#v expected %#v",
				testCase.senderIp,
				testCase.tlsMode,
				testCase.address,
				testCase.beforeSession,
				testCase.afterSession,
			)
		}
	}
}

func TestRcptTo(t *testing.T) {
	type testCase struct {
		maxRcpts      int
		address       string
		beforeSession session
		afterSession  session
		response      response
	}

	testCases := []testCase{
		// ok rcpt
		{
			1,
			"<alice@localhost>",
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "someone",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "someone",
				recipients: []string{"alice@localhost"},
				body:       "",
			},
			resOk,
		},

		// invalid sequence
		{
			2,
			"<alice@localhost>",
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "",
			},
			resInvalidSequence,
		},

		// max rcpts
		{
			1,
			"<alice@localhost>",
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "someone",
				recipients: []string{"alice@localhost"},
				body:       "",
			},
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "someone",
				recipients: []string{"alice@localhost"},
				body:       "",
			},
			resTooManyRcpts,
		},
	}

	for _, testCase := range testCases {
		res := testCase.beforeSession.handleRcptTo(testCase.maxRcpts, testCase.address)
		if !res.equals(testCase.response) {
			t.Fatalf("session.handleRcptTo(%#v, %#v) = %#v expected %#v", testCase.maxRcpts, testCase.address, res, testCase.response)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleRcptTo(%#v, %#v) resulted in session %#v expected %#v", testCase.maxRcpts, testCase.address, testCase.beforeSession, testCase.afterSession)
		}
	}
}

func TestHandleData(t *testing.T) {
	type testCase struct {
		beforeSession session
		afterSession  session
		response      response
	}

	testCases := []testCase{

		// ok
		{
			session{
				flags:      0,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "",
			},
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "",
			},
			resStartMail,
		},

		// invalid sequence
		{
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "",
			},
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "",
			},
			resInvalidSequence,
		},
	}

	for _, testCase := range testCases {
		res := testCase.beforeSession.handleData()
		if !res.equals(testCase.response) {
			t.Fatalf("session.handleData() = %#v expected %#v", res, testCase.response)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleData() resulted in session %#v expected %#v", testCase.beforeSession, testCase.afterSession)
		}
	}
}

func TestHandleBody(t *testing.T) {
	type testCase struct {
		maxMsgSize    int
		msg           string
		beforeSession session
		afterSession  session
		response      response
		finished      bool
	}

	testCases := []testCase{
		// ok
		{
			100,
			"Here is another test sentence.\n\r",
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "Here is a test sentence.\r\n",
			},
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{""},
				body:       "Here is a test sentence.\r\nHere is another test sentence.\n\r\r\n",
			},
			response{},
			false,
		},

		// ok finished
		{
			100,
			".",
			session{
				flags:      sessionFlagBodyStarted,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "Here is a test sentence.\r\n",
			},
			session{
				flags:      sessionFlagBodyStarted | sessionFlagBodyFinished,
				helloFrom:  "",
				mailFrom:   "",
				recipients: []string{},
				body:       "Here is a test sentence.\r\n",
			},
			resOk,
			true,
		},
	}

	for _, testCase := range testCases {
		finished, res := testCase.beforeSession.handleBody(testCase.maxMsgSize, testCase.msg)
		if !res.equals(testCase.response) || finished != testCase.finished {
			t.Fatalf("session.handleBody(%#v, %#v) = %#v, %#v expected %#v, %#v", testCase.maxMsgSize, testCase.msg, testCase.finished, res, finished, testCase.response)
		}
		if !testCase.beforeSession.equals(testCase.afterSession) {
			t.Fatalf("session.handleBody(%#v, %#v) resulted in session %#v expected %#v", testCase.maxMsgSize, testCase.msg, testCase.beforeSession, testCase.afterSession)
		}
	}
}
