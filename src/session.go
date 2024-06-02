package smtpin

import (
	"net"
	"strings"
)

/*
since smtp requests are not sent all at once
this is basically a struct to keep track of an
in-propgress smtp request
*/
type session struct {
	senderIp      net.IP
	senderAddr    net.Addr
	saidHello     bool
	extended      bool
	helloFrom     string
	mailFrom      string
	recipients    []string
	bodyStarted   bool
	body          string
	bodyCompleted bool
	listenCfg     *ListenConfig
	startedTls    bool // true if STARTTLS has run (NOTE: not true if implicit tls)
	resEhlo       response
}

/*
returns true if tls is off AND tls is required by the listen cfg
*/
func (s *session) needsTls() bool {
	return s.listenCfg.RequireTls && !(s.listenCfg.ImplicitTls || s.startedTls)
}

// true is body has started and not terminated
func (s *session) inBody() bool {
	return s.bodyStarted && !s.bodyCompleted
}

func (s *session) fillDefaultVals() {
	s.saidHello = false
	s.extended = false
	s.helloFrom = ""
	s.mailFrom = ""
	s.recipients = []string{}
	s.bodyStarted = false
	s.body = ""
	s.bodyCompleted = false
}

func (s *session) handleReq(req string) response {

	// handle body messages
	if s.inBody() {
		return s.handleBody(req)
	}

	// translate req into space-seperated args
	args := argSplit(req)
	argsLen := len(args)
	if argsLen == 0 {
		return resNoop
	}

	cmd := strings.ToUpper(args[0])
	if cmd == cmdMail && argsLen > 1 && strings.ToUpper(args[1]) == "FROM" {
		return s.handleMailFrom(req, argsLen)
	} else if cmd == cmdRcpt && argsLen > 1 && strings.ToUpper(args[1]) == "TO" {
		return s.handleRcptTo(req, argsLen)
	}
	switch cmd {
	case cmdEhlo:
		return s.handleEhlo(args, argsLen)
	case cmdHelo:
		return s.handleHelo(args, argsLen)
	case cmdData:
		return s.handleData()
	case cmdQuit:
		return resBye
	case cmdRset:
		s.fillDefaultVals()
		return resReset
	case cmdVrfy:
		return resCmdDisabled // TODO: possible future feature
	case cmdNoop:
		return resNoop
	case cmdTurn:
		return resCmdObsolete
	case cmdExpn:
		return resCmdDisabled // TODO: possible future feature
	case cmdHelp:
		return resCmdDisabled // TODO: possible future feature
	case cmdSend:
		return resCmdObsolete
	case cmdSaml:
		return resCmdObsolete
	case cmdRelay:
		return resCmdObsolete
	case cmdSoml:
		return resCmdObsolete
	case cmdTls:
		return resCmdObsolete
	case cmdStartTls:
		return resConnUpgrade
	case cmdStartSsl:
		return resCmdObsolete
	case cmdAuth:
		return resCmdDisabled
	}
	return resUnknownCmd
}

func (s *session) handleHelo(args []string, argsLen int) response {
	if s.saidHello {
		return resInvalidSequence
	}
	if argsLen != 2 || args[1] == "" {
		return resInvalidArgNum
	}
	s.helloFrom = strings.TrimSpace(args[1])
	s.saidHello = true
	return resHelo.withMsg("Hello " + s.helloFrom)
}

func (s *session) handleEhlo(args []string, argsLen int) response {
	s.extended = true
	if s.saidHello {
		return resInvalidSequence
	}
	if argsLen != 2 || args[1] == "" {
		return resInvalidArgNum
	}
	s.helloFrom = strings.TrimSpace(args[1])
	s.saidHello = true
	return s.resEhlo.withMsg("Hello " + s.helloFrom)
}

func (s *session) handleMailFrom(req string, argsLen int) response {
	if !s.saidHello || s.mailFrom != "" {
		return resInvalidSequence
	}
	if s.needsTls() {
		return resNeedTls
	}
	if argsLen < 3 {
		return resInvalidArgNum
	}
	emailFound, email := findEmailInLine(req)
	if !emailFound {
		return resCantParseAddr
	}

	if s.listenCfg.MailFromSpfHandler != nil {
		senderDomain := email[strings.Index(email, "@")+1:]
		senderIp := net.IP(s.senderIp)
		fail, err := s.listenCfg.MailFromSpfHandler(senderIp, senderDomain, email)
		if err != nil {
			return resSpfErr
		}
		if fail {
			return resSpfFail
		}
	}

	s.mailFrom = email
	return resAcceptingMailFrom.withMsg("Accepting mail from " + s.mailFrom)
}

func (s *session) handleRcptTo(req string, argsLen int) response {
	if !s.saidHello || s.mailFrom == "" {
		return resInvalidSequence
	}
	if s.needsTls() {
		return resNeedTls
	}
	if argsLen < 3 {
		return resInvalidArgNum
	}
	emailFound, email := findEmailInLine(req)
	if !emailFound {
		return resCantParseAddr
	}

	// check if domain is allowed
	// (only applies when defined in cfg)
	if s.listenCfg.Domains != nil && len(s.listenCfg.Domains) > 0 {
		domain := email[strings.Index(email, "@")+1:]
		allowed := false
		for _, allowedDomain := range s.listenCfg.Domains {
			if domain == allowedDomain {
				allowed = true
				break
			}
		}
		if !allowed {
			return resNotLocal
		}
	}

	s.recipients = append(s.recipients, email)
	return resRcptAdded.withMsg("Added recipient " + email)
}

func (s *session) handleData() response {
	if s.bodyStarted || len(s.recipients) == 0 || s.mailFrom == "" || !s.saidHello {
		return resInvalidSequence
	}
	if s.needsTls() {
		return resNeedTls
	}
	s.bodyStarted = true
	return resStartMail
}

func (s *session) handleBody(req string) response {
	s.body += req
	if strings.HasSuffix(s.body, bodyEnd) {
		s.bodyCompleted = true
		s.body = clipBody(s.body)
		mail := Mail{
			s.senderAddr,
			s.mailFrom,
			s.body,
			s.recipients,
		}
		if len(s.body) > s.listenCfg.MaxMsgSize {
			return resMsgTooBig
		}
		s.listenCfg.MailHandler(&mail)
		return resMailAccepted
	}
	return resBlank
}
