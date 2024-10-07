package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/csv"
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
	"github.com/quic-go/quic-go/internal/protocol"
)

const (
	SHA256File1kB  string = "9c93dbbf2e2f599b4369582b0b41e809e341dc608a2b49b547556f494f426840"
	SHA256File16kB string = "a64334619694d7e947cd25730edd24aeab1ba25c5cb9855241796857c5d0fff1"
	SHA256File65kB string = "437bb48b8a27e44b9bbc38d7176f447d72ef8cd691f1ae6cd9ab797a921e65e2"
	SHA256File1MB  string = "e82575190829096bf2061d311b272050e3af82851da270ba3d3f7b92e48efdf7"
)

// go run client.go -round=1 -host=22.22.22.22 -endpoint=1kB -scheme=1
// GOARCH=amd64 GOOS=linux go build -o client-linux-x86_64

func main() {
	round := flag.String("round", "", "the round at which this binary is being invoked.")
	host := flag.String("host", "", "the host to query")
	endpoint := flag.String("endpoint", "", "endpoint to query")
	scheme := flag.Int("scheme", 0, "0x0=>no FEC, 0x1=>XOR, 0x2=>Reed-Solomon")
	flag.Parse()

	if *round == "" || *host == "" || *endpoint == "" {
		log.Fatalf("not all flags are set. Round: %s, Host: %s, Endpoint: %s", *round, *host, *endpoint)
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
	}

	if *scheme == 0x1 {
		// XOR
		roundTripper.QuicConfig = &quic.Config{
			// Tracer:           qlog.DefaultTracer,
			EnableFEC:        true,
			DecoderFECScheme: protocol.XORFECScheme,
		}
	} else if *scheme == 0x2 {
		// Reed-Solomon
		roundTripper.QuicConfig = &quic.Config{
			// Tracer:           qlog.DefaultTracer,
			EnableFEC:        true,
			DecoderFECScheme: protocol.ReedSolomonFECScheme,
		}
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

	if isPing {
		buf, err := io.ReadAll(rsp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", string(buf))
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

	// stop timer
	duration := time.Since(start)

	f, err := os.OpenFile("test_results.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("could not open file: %v\n", err)
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	defer writer.Flush()

	fileInfo, err := f.Stat()
	if err != nil {
		log.Fatalf("failed to get file info: %s", err)
	}

	if fileInfo.Size() == 0 {
		headers := []string{"Scheme", "FileSize", "Round", "Duration"}
		if err := writer.Write(headers); err != nil {
			log.Fatalf("failed to write headers: %s", err)
		}
	}

	// FEC,1MB,2,3.45283s
	// NONE,1MB,3,2.4552s
	fecStr := "NONE"
	if *scheme == 0x1 {
		fecStr = "XOR"
	} else if *scheme == 0x2 {
		fecStr = "RS"
	}
	result := []string{
		fecStr, *endpoint, *round, duration.String(),
	}
	if err := writer.Write(result); err != nil {
		log.Fatalf("failed to write result: %s", err)
	}
}
