package smtpin

import "fmt"

const (
	codeReady            uint16 = 220
	codeBye              uint16 = 221
	codeOk               uint16 = 250
	codeNoVrfy           uint16 = 252
	codeStartMail        uint16 = 354
	codeActionAborted    uint16 = 451
	codeActionNotTaken   uint16 = 452
	codeSyntaxErr        uint16 = 501
	codeNotImplemented   uint16 = 502
	codeInvalidSequence  uint16 = 503
	codeEncryptionNeeded uint16 = 523
	codeNotLocal         uint16 = 551
	codeMsgTooBig        uint16 = 554
)

type ResponseFlag uint8

const (
	responseFlagEndConnection ResponseFlag = 0b10000000
	responseFlagDoNotRespond  ResponseFlag = 0b01000000
	responseFlagUpgradeToTls  ResponseFlag = 0b00100000
)

type response struct {
	flags        ResponseFlag
	statusCode   uint16
	msg          string
	extendedMsgs []string
}

func (r response) hasFlag(flag ResponseFlag) bool {
	return (r.flags & flag) == flag
}

func (r response) toMsg(res response) (resMsg string) {
	if len(res.extendedMsgs) != 0 {
		msgs := append([]string{res.msg}, res.extendedMsgs...)
		msgsLen := len(msgs)
		for i, msg := range msgs {
			if i+1 == msgsLen {
				resMsg += fmt.Sprintf("%d %s%s", res.statusCode, msg, crlf)
			} else {
				resMsg += fmt.Sprintf("%d-%s%s", res.statusCode, msg, crlf)
			}
		}
		return
	}
	resMsg = fmt.Sprintf("%d %s%s", res.statusCode, res.msg, crlf)
	return
}

var (
	resOk = response{
		0,
		codeOk,
		"OK",
		nil,
	}
	resHello = response{
		0,
		codeOk,
		"HELLO",
		nil,
	}
	resNoop = response{
		0,
		codeOk,
		"NO OPERATION",
		nil,
	}
	resRset = response{
		0,
		codeOk,
		"SESSION RESET",
		nil,
	}
	resNoVrfy = response{
		0,
		codeNoVrfy,
		"WILL NOT VERIFY",
		nil,
	}
	resSyntaxError = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR",
		nil,
	}
	resInvalidArgNum = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR - INVALID NUMBER OF ARGS",
		nil,
	}
	resUnknownVerb = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR - UNKNOWN VERB",
		nil,
	}
	resInvalidAddress = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR - INVALID ADDRESS",
		nil,
	}
	resNoEndingCrlf = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR - NO ENDING CRLF",
		nil,
	}
	resNotImplemented = response{
		0,
		codeNotImplemented,
		"NOT IMPLEMENTED",
		nil,
	}
	resObsolete = response{
		0,
		codeNotImplemented,
		"NOT IMPLEMENTED - OBSOLETE VERB",
		nil,
	}
	resBye = response{
		responseFlagEndConnection,
		codeBye,
		"GOODBYE",
		nil,
	}
	resInvalidSequence = response{
		0,
		codeInvalidSequence,
		"INVALID SEQUENCE",
		nil,
	}
	resStartMail = response{
		0,
		codeStartMail,
		"START MAIL",
		nil,
	}
	resSpfErr = response{
		0,
		codeActionAborted,
		"SPF ERROR",
		nil,
	}
	resDuplicateRcpt = response{
		0,
		codeActionNotTaken,
		"DUPLICATE RECIPIENT",
		nil,
	}
	resSpfFail = response{
		0,
		codeActionNotTaken,
		"SPF FAILED",
		nil,
	}
	resTlsUpgrade = response{
		responseFlagUpgradeToTls,
		codeReady,
		"READY FOR TLS UPGRADE",
		nil,
	}
	resTlsRequired = response{
		0,
		codeEncryptionNeeded,
		"TLS REQUIRED",
		nil,
	}
	resBodyTooBig = response{
		0,
		codeMsgTooBig,
		"MESSAGE TOO BIG",
		nil,
	}
	resTooManyRcpts = response{
		0,
		codeActionNotTaken,
		"TOO MANY RECIPIENTS",
		nil,
	}
	resHelp = response{
		0,
		codeOk,
		"HELP",
		[]string{"HELO", "EHLO", "STARTTLS", "MAIL FROM", "RCPT TO", "DATA", "RSET", "QUIT"},
	}
)
