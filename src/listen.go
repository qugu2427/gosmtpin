package smtpin

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

type ListenConfig struct {
	TlsConfig          *tls.Config        // (opt.) if emtpy RequireTls must be false
	ImplicitTls        bool               // whether to start tcp conn over tls (recomended: true)
	ListenAddr         string             // where to listen (ex: 0.0.0.0:25)
	MaxMsgSize         int                // max allowed size of email message
	Domains            []string           // (opt.) rcpt domains to accept for delivery (accept all domains if emtpy)
	GreetDomain        string             // domain witch will be introduced when smtp starts
	RequireTls         bool               // if set to true, requires either STARTTLS or implicit TLS for most smtp commands
	MaxConnections     int                // (opt.) maximum allowed connections (0 = 1000000)
	MailHandler        MailHandler        // function to handles mail
	MailFromSpfHandler MailFromSpfHandler // (opt.) function to handle MAIL FROM spf check
	LogErrorHandler    LogErrorHandler    // (opt.) function to handle non-fatal errors
}

/*
locally computed listener info
which is not defined in the cfg
*/
type listenInfo struct {
	resGreeting     response
	resEhlo         response
	connectionCount int
}

// builds either a tcp or tls net.Listener given cfg
func buildListener(cfg *ListenConfig) (listener net.Listener, err error) {
	if cfg.TlsConfig == nil {
		if cfg.ImplicitTls {
			err = fmt.Errorf("tls is required, but tls config is nil")
		} else {
			listener, err = net.Listen("tcp", cfg.ListenAddr)
		}
	} else {
		if cfg.ImplicitTls {
			listener, err = tls.Listen("tcp", cfg.ListenAddr, cfg.TlsConfig)
		} else {
			listener, err = net.Listen("tcp", cfg.ListenAddr)
		}
	}
	return
}

// computes local listener info based on cfg
func buildListenInfo(cfg *ListenConfig) (info listenInfo) {
	info = listenInfo{}
	info.resGreeting = response{
		true,
		true,
		false,
		codeReady,
		fmt.Sprintf("%s ESMTP Service Ready", cfg.GreetDomain),
		nil,
	}
	info.resEhlo = response{
		true,
		true,
		false,
		codeOk,
		"Hello",
		[]string{
			"PIPELINING",
			fmt.Sprintf("SIZE %d", cfg.MaxMsgSize),
		},
	}
	if !cfg.ImplicitTls && cfg.RequireTls {
		info.resEhlo.extendedMsgs = append(info.resEhlo.extendedMsgs, "STARTTLS")
	}
	if cfg.MaxConnections < 1 {
		cfg.MaxConnections = 1000000
	}
	return
}

func Listen(cfg ListenConfig) (err error) {
	listener, err := buildListener(&cfg)
	if err != nil {
		return err
	}

	defer listener.Close()

	listenInfo := buildListenInfo(&cfg)

	for {
		conn, err := listener.Accept()
		if err != nil {
			cfg.LogErrorHandler(fmt.Errorf("failed to accept connection: " + err.Error()))
			continue
		}
		err = conn.SetDeadline(time.Now().Add(5 * 60 * time.Second))
		if err != nil {
			return fmt.Errorf("failed to set deadline (%s)", err)
		}
		go handleConn(&conn, &cfg, &listenInfo)
	}
}

func handleConn(conn *net.Conn, cfg *ListenConfig, listenInfo *listenInfo) {
	defer func() {
		(*conn).Close()
		listenInfo.connectionCount--
	}()

	// check connection limit
	listenInfo.connectionCount++
	if listenInfo.connectionCount > cfg.MaxConnections {
		_ = sendRes(conn, resListenerFull)
		return
	}

	// initialize smtp session
	s := session{}
	s.resEhlo = listenInfo.resEhlo
	s.listenCfg = cfg
	s.senderAddr = (*conn).RemoteAddr()
	s.senderIp = (*conn).RemoteAddr().(*net.TCPAddr).IP
	s.fillDefaultVals()

	// greet the client
	err := sendRes(conn, listenInfo.resGreeting)
	if err != nil {
		cfg.LogErrorHandler(fmt.Errorf("failed to greet client: " + err.Error()))
		return
	}

	// each packet
	for {
		pktBuffer := make([]byte, tcpPktSize)
		pktSize, err := (*conn).Read(pktBuffer)
		if err != nil {
			cfg.LogErrorHandler(fmt.Errorf("failed to read packet: " + err.Error()))
			return
		}
		pkt := string(pktBuffer[:pktSize])
		keepAlive := handlePkt(conn, cfg, &s, pkt)
		if !keepAlive {
			return
		}
	}
}

/*
	handle each packet as a string

	this exists to allow for pipelining (i.e each pkt having multiple requests)
*/
func handlePkt(conn *net.Conn, cfg *ListenConfig, s *session, pkt string) (keepAlive bool) {
	var responses []response

	// logic to decide how crlfs should be handled
	// (pipelining, body, etc.)
	if s.inBody() {
		responses = append(responses, s.handleReq(pkt))
	} else if !strings.HasSuffix(pkt, crlf) {
		responses = []response{resInvalidCrlf}
	} else {
		pkt = strings.TrimSuffix(pkt, crlf)
		requests := strings.Split(pkt, crlf)
		for _, request := range requests {
			responses = append(responses, s.handleReq(request))
		}
	}

	var err error
	for _, response := range responses {
		if !response.respond {
			continue
		}

		// upgrade connection to tls (STARTTLS request)
		if response.upgradeToTls {
			if cfg.TlsConfig == nil {
				response = resNoTls
			} else if cfg.ImplicitTls {
				response = resTlsAlreadyEnabled
			} else {
				err := sendRes(conn, resConnUpgrade)
				if err != nil {
					cfg.LogErrorHandler(fmt.Errorf("failed to send starttls response: " + err.Error()))
					return false
				}
				tlsConn := tls.Server(*conn, cfg.TlsConfig)
				err = tlsConn.Handshake()
				if err != nil {
					response = resFailedTls
					cfg.LogErrorHandler(fmt.Errorf("failed to start tls: " + err.Error()))
					return false
				}
				*conn = net.Conn(tlsConn)
				s.fillDefaultVals()
				s.startedTls = true
				response = resBlank
			}
		}

		err = sendRes(conn, response)
		if err != nil {
			cfg.LogErrorHandler(fmt.Errorf("failed to send response: " + err.Error()))
			return false
		} else if !response.keepAlive {
			(*conn).Close()
			return false
		}
	}
	return true
}

func sendRes(conn *net.Conn, res response) (err error) {
	if !res.respond {
		return
	}
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
		}
		return
	}
	resMsg := fmt.Sprintf("%d %s%s", res.statusCode, res.msg, crlf)
	_, err = (*conn).Write([]byte(resMsg))
	return
}
