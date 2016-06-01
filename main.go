// +build ignore

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
  "strings"
  "time"

	"github.com/gorilla/websocket"
)
var sipDialer = websocket.Dialer{
  Subprotocols:    []string{"sip"},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

  c, _, err := sipDialer.Dial(*addr, nil)
  if err != nil {
    log.Fatal("dial:", err)
  }

  defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
        os.Exit(1)
				return
			}
			log.Printf("recv: %s", message)
      if (strings.Contains(string(message), "SIP/2.0 200 OK")) {
        os.Exit(0)
      } else {
        os.Exit(1)
      }
		}
	}()

// TODO: generate random ids here
var options = `OPTIONS sip:user@conf.com SIP/2.0
Via: SIP/2.0/WSS 81okseq92jb7.invalid;branch=z9hG4bK5964427
To: <sip:ba_user@none>
From: <sip:anonymous.8scs48@anonymous.invalid>;tag=fql2c8mlg3
Call-ID: gukjbo9l8s9c517q98n3
CSeq: 63104 OPTIONS
Content-Length: 0

` // two newlines signal end of request

  err = c.WriteMessage(websocket.TextMessage, []byte(options))
  if err != nil {
    log.Println("write err:", err)
    return
  }
  log.Println("write:", options)

  for {
    select {
    case <-time.After(15*time.Second):
      log.Println("read timeout")
      os.Exit(1)
    case <-interrupt:
      log.Println("interrupt")
      // To cleanly close a connection, a client should send a close
      // frame and wait for the server to close the connection.
      err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
      if err != nil {
        log.Println("write close:", err)
        return
      }
      //select {
      //case <-done:
      //}
      c.Close()
      return
    }
  }
}
