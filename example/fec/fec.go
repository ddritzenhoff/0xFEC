package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
)

const addr = "localhost:4242"

var sent chan struct{} = make(chan struct{})

// We start a server echoing data on the first stream the client opens,
// then connect with a client, send the message, and wait for its receipt.
func main() {
	go func() { log.Fatal(echoServer()) }()

	err := clientMain()
	if err != nil {
		panic(err)
	}
}

// Start a server that echos all data on the first stream opened by the client
func echoServer() error {
	cfg := quic.Config{
		EnableFEC:       true,
		EnableDatagrams: true,
	}
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), &cfg)
	if err != nil {
		return err
	}
	defer listener.Close()

	conn, err := listener.Accept(context.Background())
	if err != nil {
		return err
	}
	for {
		<-sent
		data, err := conn.ReceiveDatagram(context.Background())
		if err != nil {
			fmt.Println("hit an error")
			break
		}
		fmt.Printf("Server: Got %s\n", data)
	}

	return err
}

func clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-fec-example"},
	}
	cfg := quic.Config{
		EnableFEC:       true,
		EnableDatagrams: true,
		Tracer:          qlog.DefaultTracer,
	}

	A := []byte(strings.Repeat("A", 1500))
	B := []byte(strings.Repeat("B", 1500))

	conn, err := quic.DialAddr(context.Background(), addr, tlsConf, &cfg)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "")

	// It seems like a realistic max datagram size is
	err = conn.SendDatagramWithFEC(A[:1000])
	if err != nil {
		return err
	}

	sent <- struct{}{}

	err = conn.SendDatagramWithFEC(B[:1000])
	if err != nil {
		return err
	}

	sent <- struct{}{}

	time.Sleep(1 * time.Second)
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
		NextProtos:   []string{"quic-fec-example"},
	}
}
