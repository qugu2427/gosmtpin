package smtpin

const (
	codeReady            uint16 = 220
	codeBye              uint16 = 221
	codeAuthSuccessfull  uint16 = 235
	codeOk               uint16 = 250
	codeStartMail        uint16 = 354
	codeActionAborted    uint16 = 451
	codeAuthFailure      uint16 = 454
	codeSyntaxErr        uint16 = 500
	codeNotImplemented   uint16 = 502
	codeInvalidSequence  uint16 = 503
	codeUnrecognizedAuth uint16 = 504
	codeEncryptionNeeded uint16 = 523
	codeAuthFailed       uint16 = 535
	codeActionNotTaken   uint16 = 550
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

var (
	resOk = response{
		0,
		codeOk,
		"OK",
		nil,
	}
	resSyntaxError = response{
		0,
		codeSyntaxErr,
		"SYNTAX ERROR",
		nil,
	}
	resNotImplemented = response{
		0,
		codeNotImplemented,
		"NOT IMPLEMENTED",
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
		responseFlagEndConnection,
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
)
