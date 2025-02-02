package smtpin

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

type reqPair struct {
	request  string
	response string
}

func TestBasicSequences(t *testing.T) {
	sequences := [][]reqPair{
		{
			{"", "220 localhost ESMTP SERVICE READY\r\n"},
			{"HELO test1\r\n", "250 HELLO\r\n"},
			{"MAIL FROM:<bob@colorado.edu>\r\n", "250 OK\r\n"},
			{"RCPT TO:<alice@colorado.edu>\r\n", "250 OK\r\n"},
			{"DATA\r\n", "354 START MAIL\r\n"},
			{"abc\r\n", ""},
			{"123\r\n", ""},
			{".\r\n", "250 OK\r\n"},
			{"RSET\r\n", "250 SESSION RESET\r\n"},
			{"EHLO test2\r\n", "250-HELLO\r\n250-PIPELINING\r\n250-SIZE 1024\r\n250 STARTTLS\r\n"},
			{"MAIL FROM:<bob@colorado.edu>\r\n", "250 OK\r\n"},
			{"RCPT TO:<alice@colorado.edu>\r\n", "250 OK\r\n"},
			{"DATA\r\nabc\r\n123\r\n.\r\n", "354 START MAIL\r\n250 OK\r\n"},
			{"QUIT\r\n", "221 GOODBYE\r\n"},
		},
	}

	listener := Listener{
		TlsMode:     TlsModeNone,
		TlsConfig:   nil,
		Host:        "0.0.0.0",
		Port:        2525,
		MaxRcpts:    100,
		MaxMsgSize:  1024,
		HandleError: func(err error) {},
		HandleMail:  func(mail *Mail) {},
		HandleSpf:   nil,
		Domain:      "localhost",
	}

	testSequences(t, listener, sequences)
}

func testSequences(t *testing.T, listener Listener, sequences [][]reqPair) {
	go func() {
		err := listener.Listen()
		if err != nil {
			fmt.Printf("error during listener.Listen(): %s", err)
			os.Exit(1)
		}
	}()

	time.Sleep(3 * time.Second)

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", 2525))
	if err != nil {
		t.Fatalf("error during dial tcp: %s", err)
		return
	}

	defer conn.Close()

	for _, sequence := range sequences {
		for _, reqPair := range sequence {

			_, err = conn.Write([]byte(reqPair.request))
			if err != nil {
				t.Fatalf("error sending request: %s", err)
				return
			}

			if reqPair.response != "" {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					t.Fatalf("error reading response: %s", err)
					return
				}
				res := string(buf[:n])

				if res != reqPair.response {
					t.Fatalf("%#v got res %#v, expected res %#v", reqPair.request, res, reqPair.response)
					return
				}
			}
		}
	}
}
