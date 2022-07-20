// MIT License
//
// (C) Copyright 2020-2022 Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/namsral/flag"
)

var (
	tlsEnabled = flag.Bool("tls_enabled", false, "Listen with TLS?")
	tlsCert    = flag.String("tls_cert", "tls.crt", "Path to the TLS Certificate")
	tlsKey     = flag.String("tls_key", "tls.key", "Path to the TLS Key")
)

func parseRequest(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("ERROR: Could not read body!")
	}

	bodyString := string(bodyBytes)
	var prettyJSON bytes.Buffer
	jsonErr := json.Indent(&prettyJSON, bodyBytes, "", "\t")
	if jsonErr != nil {
		log.Println("ERROR: Could not parse JSON payload! Full payload:")

		if bodyString == "" {
			log.Println("\t<null>")
		} else {
			lines := strings.Split(bodyString, "\n")
			for _, line := range lines {
				log.Printf("\t%s\n", line)
			}
		}

		// Best to let the client know they dun goofed.
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("JSON payload malformed!"))
		if err != nil {
			log.Println("ERROR: Could not write that JSON was malformed!")
		}
	} else {
		log.Println(string(prettyJSON.Bytes()) + "\n") // Debug
	}

}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "HTTPs example")
		fmt.Fprintln(os.Stderr, "1. Generate HTTPS Certificate")
		fmt.Fprintln(os.Stderr, "  $ openssl req -newkey rsa:4096 \\")
		fmt.Fprintln(os.Stderr, "      -x509 -sha256 \\")
		fmt.Fprintln(os.Stderr, "      -days 1 \\")
		fmt.Fprintln(os.Stderr, "      -nodes \\")
		fmt.Fprintln(os.Stderr, "      -subj \"/C=US/ST=Minnesota/L=Bloomington/O=HPE/OU=Engineering/CN=hpe.com\" \\")
		fmt.Fprintln(os.Stderr, "      -out tls.crt \\")
		fmt.Fprintln(os.Stderr, "      -keyout tls.key")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "2. Start the echo server")
		fmt.Fprintf(os.Stderr, "  $ %s -tls_enabled\n", os.Args[0])

		fmt.Println()
		fmt.Println("HTTP example")
		fmt.Fprintln(os.Stderr, "1. Start the echo server")
		fmt.Fprintf(os.Stderr, "  $ %s", os.Args[0])
		fmt.Println()
	}

	flag.Parse()

	http.HandleFunc("/", parseRequest)
	if *tlsEnabled {
		log.Println("TLS Certificate:", *tlsCert)
		log.Println("TLS Key:        ", *tlsKey)
		log.Println("Listening with TLS on port 443")
		log.Fatal(http.ListenAndServeTLS(":443", *tlsCert, *tlsKey, nil))
	} else {
		log.Println("Listening without TLS on port 80")
		log.Fatal(http.ListenAndServe(":80", nil))
	}

}
