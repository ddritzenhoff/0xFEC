package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"strings"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/internal/protocol"
)

const addr = "localhost:4242"

const messageLen = 2000

// We start a server echoing data on the first stream the client opens,
// then connect with a client, send the message, and wait for its receipt.
func main() {
	done := make(chan struct{})

	go func() { log.Fatal(echoServer(done)) }()

	err := clientMain(done)
	if err != nil {
		panic(err)
	}
}

// Start a server that echos all data on the first stream opened by the client
func echoServer(done chan<- struct{}) error {
	quicConf := &quic.Config{
		EnableFEC:        true,
		DecoderFECScheme: protocol.XORFECScheme,
	}
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), quicConf)
	if err != nil {
		return err
	}
	defer listener.Close()

	conn, err := listener.Accept(context.Background())
	if err != nil {
		return err
	}

	stream, err := conn.AcceptUniStream(context.Background())
	if err != nil {
		panic(err)
	}

	buf := make([]byte, messageLen)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return err
	}
	fmt.Printf("Server: %x\n", sha256.Sum256(buf))

	done <- struct{}{}

	return err
}

func clientMain(done <-chan struct{}) error {
	quicConf := &quic.Config{
		EnableFEC:        true,
		DecoderFECScheme: protocol.XORFECScheme,
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	conn, err := quic.DialAddr(context.Background(), addr, tlsConf, quicConf)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenUniStreamSyncWithFEC(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	A := []byte(strings.Repeat("A", messageLen))

	fmt.Printf("Client: %x\n", sha256.Sum256(A))
	_, err = stream.Write([]byte(A))
	if err != nil {
		return err
	}

	<-done

	return nil
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
