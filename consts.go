package smtpin

const (
	crlf    string = "\r\n"
	bodyEnd string = "\r\n.\r\n"
)

const (
	codeReady          uint16 = 220
	codeBye            uint16 = 221
	codeOk             uint16 = 250
	codeStartMail      uint16 = 354
	codeActionAborted  uint16 = 451
	codeAuthFailure    uint16 = 454
	codeSyntaxErr      uint16 = 500
	codeNotImplemented uint16 = 502
	codeActionNotTaken uint16 = 550
	codeNotLocal       uint16 = 551
	codeMsgTooBig      uint16 = 554
)

const (
	cmdHelo     string = "HELO"
	cmdEhlo     string = "EHLO"
	cmdMail     string = "MAIL"
	cmdRcpt     string = "RCPT"
	cmdData     string = "DATA"
	cmdQuit     string = "QUIT"
	cmdRset     string = "RSET"
	cmdVrfy     string = "VRFY"
	cmdNoop     string = "NOOP"
	cmdTurn     string = "TURN"
	cmdExpn     string = "EXPN"
	cmdHelp     string = "HELP"
	cmdSend     string = "SEND"
	cmdSaml     string = "SAML"
	cmdSoml     string = "SOML"
	cmdTls      string = "TLS"
	cmdStartTls string = "STARTTLS"
	cmdStartSsl string = "STARTSSL"
	cmdRelay    string = "RELAY"
	cmdAuth     string = "AUTH"
)
