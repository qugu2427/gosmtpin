package smtpin

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

type ListenerTlsMode uint8

var ConnectionTimeout time.Duration = 5 * 60 * time.Second
var PrintTraceLogs bool = true

const (
	TlsModeImplicit         ListenerTlsMode = iota // connection will start over tls (recomended)
	TlsModeStartTls                                // connection MUST upgrade to tls with STARTTLS command
	TlsModeStartTlsOptional                        // connection MAY upgrade to tls with STARTTLS command (insecure)
	TlsModeNone                                    // connection cannot be upgraded to tls
)

const (
	prefixHelo     string = "HELO"
	prefixEhlo     string = "EHLO"
	prefixMailFrom string = "MAIL FROM:"
	prefixRcptTo   string = "RCPT TO:"
	prefixData     string = "DATA"
	prefixQuit     string = "QUIT"
	prefixRset     string = "RSET"
	prefixVrfy     string = "VRFY"
	prefixNoop     string = "NOOP"
	prefixTurn     string = "TURN"
	prefixExpn     string = "EXPN"
	prefixHelp     string = "HELP"
	prefixSend     string = "SEND"
	prefixSaml     string = "SAML"
	prefixSoml     string = "SOML"
	prefixTls      string = "TLS"
	prefixStartTls string = "STARTTLS"
	prefixStartSsl string = "STARTSSL"
	prefixRelay    string = "RELAY"
	prefixAuth     string = "AUTH"
)

type ErrorHandler = func(err error)
type SpfHandler = func(senderIp net.IP, senderDomain, senderSender string) (fail bool, err error)
type MailHandler = func(mail *Mail)

type Mail struct {
	Helo       string
	SenderIp   net.IP
	MailFrom   string
	Data       string
	Recipients []string
}

type Listener struct {
	TlsMode     ListenerTlsMode // tls mode (recomended: Implicit)
	TlsConfig   *tls.Config     // (opt.) tls config
	Host        string          // (ex: 0.0.0.0)
	Port        uint16          // (ex: 25)
	MaxRcpts    int             // (opt.) maximum allows rcpts (<0 = infinity)
	MaxMsgSize  int             // max allowed size of email message
	HandleError ErrorHandler    // (opt.) handle non fatal errors
	HandleSpf   SpfHandler      // (opt.) handle spf information from mail from
	HandleMail  MailHandler     // handle end mail
	Domain      string          // domain to accept on behalf of
}

// builds either a tcp or tls net.Listener
func (listener *Listener) build() (netListener net.Listener, err error) {
	if listener.TlsConfig == nil && listener.TlsMode != TlsModeNone {
		err = fmt.Errorf("tls config must be specified")
	} else if listener.TlsMode == TlsModeImplicit {
		netListener, err = tls.Listen("tcp", fmt.Sprintf("%s:%d", listener.Host, listener.Port), listener.TlsConfig)
	} else {
		netListener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", listener.Host, listener.Port))
	}
	return
}

func (listener *Listener) Listen() (err error) {
	netListener, err := listener.build()
	if err != nil {
		return err
	}

	defer netListener.Close()

	for {
		conn, err := netListener.Accept()
		if err != nil {
			listener.HandleError(fmt.Errorf("failed to accept connection: %s", err.Error()))
			continue
		}
		err = conn.SetDeadline(time.Now().Add(ConnectionTimeout))
		if err != nil {
			return fmt.Errorf("failed to set deadline (%s)", err)
		}
		go listener.handleConn(&conn)
	}
}

func (listener *Listener) handleConn(conn *net.Conn) {
	defer func() {
		remoteAddr := (*conn).RemoteAddr().String()
		(*conn).Close()
		printTraceLog("%s -- closed connection\n", remoteAddr)
	}()

	printTraceLog("%s -- started connection\n", (*conn).RemoteAddr().String())

	// initialize smtp session
	session := createNewSession()
	if listener.TlsMode == TlsModeImplicit {
		session.addFlag(sessionFlagTlsEnabled)
	}

	// greet the client
	err := sendRes(conn, response{
		statusCode:   codeReady,
		msg:          listener.Domain + " ESMTP SERVICE READY",
		extendedMsgs: nil,
	})
	if err != nil {
		listener.HandleError(fmt.Errorf("failed to greet client: %s", err.Error()))
		return
	}

	// each packet
	pktBuffer := make([]byte, 1024)
	for {
		var pktSize int
		pktSize, err = (*conn).Read(pktBuffer)
		if err != nil {
			listener.HandleError(fmt.Errorf("failed to read packet: %s", err.Error()))
			return
		}
		pkt := string(pktBuffer[:pktSize])
		if strings.HasSuffix(pkt, crlf) {
			msgs := strings.Split(strings.TrimSuffix(pkt, crlf), crlf)
			for _, msg := range msgs {
				keepAlive := listener.handleMsg(conn, &session, msg)
				if !keepAlive {
					return
				}
			}
		} else {
			listener.HandleError(fmt.Errorf("packet does not end in crlf"))
			return
		}
	}
}

func (listener *Listener) handleMsg(conn *net.Conn, session *session, msg string) (keepAlive bool) {
	var res response
	printTraceLog("%s -> %#v\n", (*conn).RemoteAddr().String(), msg+crlf)
	if session.isInBody() {
		var finished bool
		finished, res = session.handleBody(listener.MaxMsgSize, msg)
		if !finished {
			keepAlive = true
			return
		}
		listener.HandleMail(&Mail{
			Helo:       session.helloFrom,
			SenderIp:   (*conn).RemoteAddr().(*net.TCPAddr).IP,
			MailFrom:   session.mailFrom,
			Data:       session.body,
			Recipients: session.recipients,
		})
	} else {
		res = listener.handleCmd(conn, session, msg)
	}
	err := sendRes(conn, res)
	if err != nil {
		listener.HandleError(fmt.Errorf("failed send response: %s", err.Error()))
		return
	}
	if !res.hasFlag(responseFlagEndConnection) {
		keepAlive = true
		return
	}
	if res.hasFlag(responseFlagUpgradeToTls) {
		tlsConn := tls.Server(*conn, listener.TlsConfig)
		err = tlsConn.Handshake()
		if err != nil {
			listener.HandleError(fmt.Errorf("failed starttls handshake: %s", err.Error()))
			return
		}
		*session = createNewSession()
		*conn = net.Conn(tlsConn)
	}
	return
}

func (listener *Listener) handleCmd(conn *net.Conn, session *session, cmd string) (res response) {
	cmd = strings.ToUpper(cmd)
	words := strings.Split(cmd, " ")
	wordsLen := len(words)
	if strings.HasPrefix(cmd, prefixHelo) && wordsLen > 1 {
		domain := words[1]
		return session.handleHelo(domain)
	} else if strings.HasPrefix(cmd, prefixEhlo) && wordsLen > 1 {
		domain := words[1]
		return session.handleEhlo(domain, listener.MaxMsgSize)
	} else if strings.HasPrefix(cmd, prefixMailFrom) && wordsLen > 1 {
		cmdArgsMsg := strings.TrimSpace(strings.TrimPrefix(cmd, prefixMailFrom))
		cmdArgs := strings.Split(cmdArgsMsg, " ")
		address := cmdArgs[0]
		return session.handleMailFrom((*conn).RemoteAddr().(*net.TCPAddr).IP, listener.HandleSpf, listener.TlsMode, address)
	} else if strings.HasPrefix(cmd, prefixRcptTo) && wordsLen > 1 {
		cmdArgsMsg := strings.TrimSpace(strings.TrimPrefix(cmd, prefixRcptTo))
		cmdArgs := strings.Split(cmdArgsMsg, " ")
		address := cmdArgs[0]
		return session.handleRcptTo(int(listener.MaxRcpts), address)
	} else if strings.HasPrefix(cmd, prefixData) {
		return session.handleData()
	} else if strings.HasPrefix(cmd, prefixQuit) {
		return resBye
	} else if strings.HasPrefix(cmd, prefixRset) {
		*session = createNewSession()
		return resOk
	} else if strings.HasPrefix(cmd, prefixStartTls) {
		return resTlsUpgrade
	} else if strings.HasPrefix(cmd, prefixNoop) {
		return resOk
	} else if strings.HasPrefix(cmd, prefixVrfy) ||
		strings.HasPrefix(cmd, prefixTurn) ||
		strings.HasPrefix(cmd, prefixExpn) ||
		strings.HasPrefix(cmd, prefixHelp) ||
		strings.HasPrefix(cmd, prefixSend) ||
		strings.HasPrefix(cmd, prefixSaml) ||
		strings.HasPrefix(cmd, prefixTls) ||
		strings.HasPrefix(cmd, prefixStartSsl) ||
		strings.HasPrefix(cmd, prefixRelay) ||
		strings.HasPrefix(cmd, prefixAuth) {
		return resNotImplemented
	}
	return resSyntaxError
}

func sendRes(conn *net.Conn, res response) (err error) {
	if len(res.extendedMsgs) != 0 {
		msgs := append([]string{res.msg}, res.extendedMsgs...)
		msgsLen := len(msgs)
		for i, msg := range msgs {
			var resMsg string
			if i+1 == msgsLen {
				resMsg = fmt.Sprintf("%d %s%s", res.statusCode, msg, crlf)
			} else {
				resMsg = fmt.Sprintf("%d-%s%s", res.statusCode, msg, crlf)
			}
			_, err = (*conn).Write([]byte(resMsg))
			if err != nil {
				return
			}
			printTraceLog("%s <- %#v\n", (*conn).RemoteAddr().String(), resMsg)
		}
		return
	}
	resMsg := fmt.Sprintf("%d %s%s", res.statusCode, res.msg, crlf)
	_, err = (*conn).Write([]byte(resMsg))
	printTraceLog("%s <- %#v\n", (*conn).RemoteAddr().String(), resMsg)
	return
}
