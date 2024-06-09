package main

import (
	"fmt"
	"smtpin"
)

func main() {

	errHandler := func(err error) {
		fmt.Println("Err: ", err)
	}

	mailHandler := func(mail *smtpin.Mail) {
		fmt.Println("Revieved mail: ", mail)
	}

	cfg := smtpin.ListenConfig{
		ImplicitTls:     false,
		ListenAddr:      "0.0.0.0:25",
		MaxMsgSize:      10000,
		GreetDomain:     "local-host.com",
		RequireTls:      false,
		MaxConnections:  3,
		MaxRcpts:        3,
		MailHandler:     mailHandler,
		LogErrorHandler: errHandler,
	}

	err := smtpin.Listen(cfg)
	if err != nil {
		panic(err)
	}
}
