package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const (
	SHA256File1kB  string = "9c93dbbf2e2f599b4369582b0b41e809e341dc608a2b49b547556f494f426840"
	SHA256File16kB string = "a64334619694d7e947cd25730edd24aeab1ba25c5cb9855241796857c5d0fff1"
	SHA256File65kB string = "437bb48b8a27e44b9bbc38d7176f447d72ef8cd691f1ae6cd9ab797a921e65e2"
	SHA256File1MB  string = "e82575190829096bf2061d311b272050e3af82851da270ba3d3f7b92e48efdf7"
)

// go run client.go -round=1 -file=output.txt -host=localhost -endpoint=1kB
// GOARCH=amd64 GOOS=linux go build -o client-linux-x86_64

func main() {
	round := flag.String("round", "", "the round at which this binary is being invoked.")
	outputFile := flag.String("file", "", "file to write the download completion time")
	host := flag.String("host", "", "the host to query")
	endpoint := flag.String("endpoint", "", "endpoint to query")

	flag.Parse()

	if *round == "" || *outputFile == "" || *host == "" || *endpoint == "" {
		log.Fatalf("not all flags are set. Round: %s, Output file: %s, Host: %s, Endpoint: %s", *round, *outputFile, *host, *endpoint)
	}

	isPing := false
	var expectedDigest string
	switch *endpoint {
	case "1kB":
		expectedDigest = SHA256File1kB
	case "16kB":
		expectedDigest = SHA256File16kB
	case "65kB":
		expectedDigest = SHA256File65kB
	case "1MB":
		expectedDigest = SHA256File1MB
	case "ping":
		isPing = true
	default:
		log.Fatalf("addr does not have a valid endpoint: %s", *endpoint)
	}

	addr := fmt.Sprintf("https://%s/%s", *host, *endpoint)

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		QuicConfig: &quic.Config{
			// Tracer:           qlog.DefaultTracer,
			// EnableFEC:        true,
			// DecoderFECScheme: protocol.XORFECScheme,
		},
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	// start timer
	start := time.Now()

	// make the actual request
	rsp, err := hclient.Get(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer rsp.Body.Close()

	// stop timer
	duration := time.Since(start)

	if isPing {
		buf, err := io.ReadAll(rsp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %s\n", string(buf), duration.String())
		return
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, rsp.Body); err != nil {
		log.Fatalf("Error reading body: %v\n", err)
	}
	digest := hex.EncodeToString(hash.Sum(nil))

	if digest != expectedDigest {
		log.Fatal("received digest not equal to expected digest")
	}

	f, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("could not open file: %v\n", err)
	}
	defer f.Close()

	res := fmt.Sprintf("round %s DCT for %s: %s\n", *round, *endpoint, duration.String())
	if _, err := f.WriteString(res); err != nil {
		log.Fatalf("could not write to file: %v\n", err)
	}
	fmt.Print(res)
}
