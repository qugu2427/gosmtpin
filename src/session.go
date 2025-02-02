package smtpin

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type SessionFlag uint8

const (
	crlf    string = "\r\n"
	bodyEnd string = "\r\n.\r\n"
)

const (
	sessionFlagSaidHello    SessionFlag = 0b10000000
	sessionFlagExtended     SessionFlag = 0b01000000
	sessionFlagBodyStarted  SessionFlag = 0b00100000
	sessionFlagBodyFinished SessionFlag = 0b00010000
	sessionFlagTlsEnabled   SessionFlag = 0b00001000
)

var (
	rgxHelo    *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9-\./_:\[\]\\]{1,255}$`)
	rgxAddress *regexp.Regexp = regexp.MustCompile(`^[^@]{1,64}@[a-zA-Z0-9-\.]{1,255}$`)
)

type session struct {
	flags      SessionFlag
	helloFrom  string
	mailFrom   string
	recipients []string
	body       string
}

func (s0 session) equals(s1 session) bool {
	if len(s0.recipients) != len(s1.recipients) {
		return false
	}

	for i, s0Rcpt := range s0.recipients {
		if s0Rcpt != s1.recipients[i] {
			return false
		}
	}

	return s0.flags == s1.flags &&
		s0.helloFrom == s1.helloFrom &&
		s0.mailFrom == s1.mailFrom &&
		s0.body == s1.body
}

func createNewSession() session {
	return session{
		0,
		"",
		"",
		nil,
		"",
	}
}

func (s *session) hasFlag(flag SessionFlag) bool {
	return (flag & s.flags) == flag
}

func (s *session) addFlag(flag SessionFlag) {
	s.flags |= flag
}

func (s *session) isInBody() bool {
	return s.hasFlag(sessionFlagBodyStarted) && !s.hasFlag(sessionFlagBodyFinished)
}

func (s *session) handleHelo(helloFrom string) (res response) {
	if s.hasFlag(sessionFlagSaidHello) {
		return resInvalidSequence
	}
	if !rgxHelo.MatchString(helloFrom) {
		return resSyntaxError
	}
	s.helloFrom = helloFrom
	s.addFlag(sessionFlagSaidHello)
	return resHello
}

func (s *session) handleEhlo(helloFrom string, maxMsgSize int) (res response) {
	if s.hasFlag(sessionFlagSaidHello) {
		return resInvalidSequence
	}
	if !rgxHelo.MatchString(helloFrom) {
		return resSyntaxError
	}
	s.helloFrom = helloFrom
	s.addFlag(sessionFlagSaidHello)
	s.addFlag(sessionFlagExtended)
	res = response{
		statusCode:   codeOk,
		msg:          resHello.msg,
		extendedMsgs: []string{"PIPELINING", fmt.Sprintf("SIZE %d", maxMsgSize)},
	}
	if !s.hasFlag(sessionFlagTlsEnabled) {
		res.extendedMsgs = append(res.extendedMsgs, "STARTTLS")
	}
	return
}

func (s *session) handleMailFrom(senderIp net.IP, handleSpf SpfHandler, tlsMode ListenerTlsMode, address string) (res response) {
	if !s.hasFlag(sessionFlagSaidHello) || s.mailFrom != "" {
		return resInvalidSequence
	}
	if !s.hasFlag(sessionFlagTlsEnabled) && tlsMode == TlsModeStartTls {
		return resTlsRequired
	}
	if address == "" {
		address = s.helloFrom
	}
	if !rgxAddress.MatchString(address) {
		return resInvalidAddress
	}
	if handleSpf != nil {
		fail, err := handleSpf(senderIp, address[strings.Index(address, "@"):], address)
		if err != nil {
			return resSpfErr
		}
		if fail {
			return resSpfFail
		}
	}
	s.mailFrom = address
	return resOk
}

func (s *session) handleRcptTo(maxRcpts int, address string) (res response) {
	if s.mailFrom == "" {
		return resInvalidSequence
	}
	if !rgxAddress.MatchString(address) && strings.ToLower(address) != "postmaster" {
		return resInvalidAddress
	}
	if len(s.recipients) >= maxRcpts && maxRcpts >= 0 {
		return resTooManyRcpts
	}
	if contains(s.recipients, address) {
		return resDuplicateRcpt
	}
	s.recipients = append(s.recipients, address)
	return resOk
}

func (s *session) handleData() (res response) {
	if len(s.recipients) == 0 || s.hasFlag(sessionFlagBodyStarted) {
		return resInvalidSequence
	}
	s.addFlag(sessionFlagBodyStarted)
	return resStartMail
}

func (s *session) handleBody(maxMsgSize int, msg string) (finished bool, res response) {
	if msg == "." {
		s.addFlag(sessionFlagBodyFinished)
		return true, resOk
	} else {
		s.body += msg + crlf
		if len(s.body) > maxMsgSize {
			return true, resBodyTooBig
		}
		return false, response{}
	}
}
