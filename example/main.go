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

	listener := smtpin.Listener{
		TlsMode:     smtpin.TlsModeNone,
		TlsConfig:   nil,
		Host:        "0.0.0.0",
		Port:        2525,
		MaxRcpts:    100,
		MaxMsgSize:  1024,
		HandleError: errHandler,
		HandleMail:  mailHandler,
		HandleSpf:   nil,
		Domain:      "localhost",
	}

	err := listener.Listen()
	if err != nil {
		panic(err)
	}
}
