package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"strings"

	"github.com/quic-go/quic-go"
)

var (
	addr       string
	messageLen int
)

func init() {
	flag.StringVar(&addr, "addr", "localhost:4242", "server address")
	flag.IntVar(&messageLen, "len", 20000, "length of the message to send")
}

// We start a server echoing data on the first stream the client opens,
// then connect with a client, send the message, and wait for its receipt.
func main() {
	flag.Parse()

	err := clientMain()
	if err != nil {
		panic(err)
	}
}

func clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-fec-example"},
	}
	conn, err := quic.DialAddr(context.Background(), addr, tlsConf, nil)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenUniStreamSyncWithFEC(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	msg := []byte(strings.Repeat("A", messageLen))

	_, err = stream.Write([]byte(msg))
	if err != nil {
		return err
	}

	fmt.Printf("%x\n", sha256.Sum256(msg))

	return nil
}
