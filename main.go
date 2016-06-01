package main

import (
  "bufio"
  "crypto/rand"
  "crypto/tls"
  "flag"
  "fmt"
  "log"
  "os"
  "os/signal"
  "strings"
  "strconv"
  "time"
  "net/url"
  "net"
  mathrand "math/rand"

  "github.com/gorilla/websocket"
)
var addr = flag.String("addr", "", "http service address")
var skipVerify = flag.Bool("skipverify", false, "skip TLS certificate verification")

const WS_OPTIONS = `OPTIONS sip:monitor@none SIP/2.0
Via: SIP/2.0/WSS 81okseq92jb7.invalid;branch=z9hG4bK5964427
To: <sip:ba_user@none>
From: <sip:anonymous.8scs48@anonymous.invalid>;tag=fql2c8mlg3
Call-ID: {{callId}}
CSeq: {{seq}} OPTIONS
Content-Length: 0

` // two newlines required to signal end of request

const TCP_OPTIONS = `OPTIONS sip:host@invalid:1738;transport=tcp SIP/2.0
Via: SIP/2.0/TCP 127.0.0.1:5066;branch=z9hG4bKr1t13cmvZDjtg
Max-Forwards: 70
From: "" <sip:monitor@invalid>
To: <sip:host@invalid;transport=tcp>
Call-ID: {{callId}}
CSeq: {{seq}} OPTIONS
Content-Length: 0

` // two newlines required to signal end of request

func randString(n int) string {
    const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, n)
    rand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

func renderRequest(options string) string {
    mathrand.Seed(time.Now().UnixNano())
    req := strings.Replace(options, "{{callId}}", randString(20), -1)
    req = strings.Replace(req, "{{seq}}", strconv.Itoa(mathrand.Intn(99999)), -1)
    req = strings.Replace(req, "\n", "\r\n", -1)
    return req
}

func main() {
  flag.Parse()
  log.SetFlags(0)

  if (len(*addr)==0) {
    flag.Usage()
    os.Exit(255)
  }

  var url, err = url.Parse(*addr)
  if err != nil {
    log.Fatal("addr:", err)
  }

  if (url.Scheme == "wss" || url.Scheme == "ws") {

    var tlsClientConfig = &tls.Config{InsecureSkipVerify: *skipVerify}
    var sipDialer = websocket.Dialer{
      Subprotocols:    []string{"sip"},
      ReadBufferSize:  1024,
      WriteBufferSize: 1024,
      TLSClientConfig: tlsClientConfig,
    }

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

    req := renderRequest(WS_OPTIONS)

    err = c.WriteMessage(websocket.TextMessage, []byte(req))
    if err != nil {
      log.Println("write err:", err)
      return
    }
    log.Println("======================")
    log.Println(req)
    log.Println("======================")

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
        select {
        case <-done:
        }
        c.Close()
        return
      }
    }
  } else if (url.Scheme == "sip" || url.Scheme == "tcp") {
    conn, err := net.Dial("tcp", url.Host)
    if err != nil {
      log.Fatal("dial:", err)
    }
    req := renderRequest(TCP_OPTIONS)
    log.Println("======================")
    log.Println(req)
    log.Println("======================")
    fmt.Fprintf(conn, req)
    status, err := bufio.NewReader(conn).ReadString('\n')
    if err != nil {
      log.Fatal("read:", err)
      os.Exit(1)
    }
    log.Println(status)
    if (strings.Contains(status, "SIP/2.0 200 OK")) {
      os.Exit(0)
    } else {
      os.Exit(1)
    }
  } else {
    log.Println("Unknown scheme:", url.Scheme);
    os.Exit(255)
  }
}

