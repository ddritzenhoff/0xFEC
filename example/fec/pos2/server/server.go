package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/internal/protocol"
)

func setupHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/1kB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/1kB")
	})
	mux.HandleFunc("/10kB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/10kB")
	})
	mux.HandleFunc("/16kB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/16kB")
	})
	mux.HandleFunc("/50kB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/50kB")
	})
	mux.HandleFunc("/65kB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/65kB")
	})
	mux.HandleFunc("/1MB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/1MB")
	})
	mux.HandleFunc("/10MB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/10MB")
	})
	mux.HandleFunc("/20MB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/20MB")
	})
	mux.HandleFunc("/30MB", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/tmp/30MB")
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	return mux
}

// GOARCH=amd64 GOOS=linux go build -o server-linux-x86_64
// go run server.go

func main() {
	fileDigests := flag.Bool("file-digests", false, "Does not start the server. Prints out the file digests")
	scheme := flag.Int("scheme", 0, "0x0=>no FEC, 0x1=>XOR, 0x2=>Reed-Solomon")
	flag.Parse()

	if *fileDigests {
		printFileDigests()
		return
	}

	server := http3.Server{
		Handler: setupHandler(),
		Addr:    ":443",
		TLSConfig: http3.ConfigureTLSConfig(&tls.Config{
			Certificates: generateTLSCertificates(),
		}),
	}

	if *scheme == 0x1 {
		// XOR
		server.QuicConfig = &quic.Config{
			// Tracer:           qlog.DefaultTracer,
			EnableFEC:        true,
			DecoderFECScheme: protocol.XORFECScheme,
		}
	} else if *scheme == 0x2 {
		// Reed-Solomon
		server.QuicConfig = &quic.Config{
			// Tracer:           qlog.DefaultTracer,
			EnableFEC:        true,
			DecoderFECScheme: protocol.ReedSolomonFECScheme,
		}
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func printFileDigests() {
	fileNames := []string{"1kB", "10kB", "16kB", "50kB", "65kB", "1MB"}
	for _, fileName := range fileNames {
		path := "./" + fileName
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", path, err)
			continue
		}
		defer file.Close()

		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			fmt.Printf("Error reading file %s: %v\n", path, err)
			continue
		}

		digest := hash.Sum(nil)
		fmt.Printf("SHA-256 digest of %s: %s\n", path, hex.EncodeToString(digest))
	}
}

// Setup a bare-bones TLS config for the server
func generateTLSCertificates() []tls.Certificate {
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
	return []tls.Certificate{tlsCert}
}
