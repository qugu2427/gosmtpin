package smtpin

import (
	"fmt"
	"net"
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

type session struct {
	flags      SessionFlag
	helloFrom  string
	mailFrom   string
	recipients []string
	body       string
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

func (s *session) handleHelo(domain string) (res response) {
	if s.hasFlag(sessionFlagSaidHello) {
		return resInvalidSequence
	}
	if !isValidHelo(domain) {
		return resSyntaxError
	}
	s.helloFrom = domain
	s.addFlag(sessionFlagSaidHello)
	return resOk
}

func (s *session) handleEhlo(domain string, maxMsgSize int) (res response) {
	if s.hasFlag(sessionFlagSaidHello) {
		return resInvalidSequence
	}
	if !isValidHelo(domain) {
		return resSyntaxError
	}
	s.helloFrom = domain
	s.addFlag(sessionFlagSaidHello)
	s.addFlag(sessionFlagExtended)
	return response{
		statusCode:   codeOk,
		msg:          resOk.msg,
		extendedMsgs: []string{"PIPELINING", "STARTTLS", fmt.Sprintf("SIZE %d", maxMsgSize)},
	}
}

func (s *session) handleMailFrom(senderIp net.IP, handleSpf SpfHandler, tlsMode ListenerTlsMode, address string) (res response) {
	if !s.hasFlag(sessionFlagSaidHello) || s.mailFrom != "" {
		return resInvalidSequence
	}
	if !s.hasFlag(sessionFlagTlsEnabled) && tlsMode == TlsModeStartTls {
		return resTlsRequired
	}
	address, isValid := extractAddress(address)
	if !isValid {
		return resSyntaxError
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
	address, isValid := extractAddress(address)
	if !isValid {
		return resSyntaxError
	}
	if len(s.recipients) >= maxRcpts && maxRcpts >= 0 {
		return resTooManyRcpts
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
