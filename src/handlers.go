package smtpin

import "net"

/*
	Handler called on MAIL FROM: <sender@domain>

	if returns fail = true, smtp exchange will terminate with "550 Spf check failed"

	if returns err = not nil, smtp exchange will terminate with "451 Spf error"

	(optional handler)
*/
type MailFromSpfHandler = func(senderIp net.IP, senderDomain, senderSender string) (fail bool, err error)

/*
	Handler called when smtp exchange is completed

	(required handler)
*/
type MailHandler = func(mail *Mail)

type Mail struct {
	SenderAddr net.Addr
	MailFrom   string
	Raw        string
	Recipients []string
}

/*
	Handler called when *non-fatal* error occurs, such as a failed connection or invalid smtp request.

	(optional handler)
*/
type LogErrorHandler = func(err error)
