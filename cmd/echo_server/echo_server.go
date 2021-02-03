// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
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
    "github.com/namsral/flag"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
)

var(
    tlsEnabled = flag.Bool("tls_enabled", false, "Listen with TLS?")
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
    flag.Parse()

    http.HandleFunc("/", parseRequest)
    if *tlsEnabled {
        log.Println("Listening with TLS on port 443")
        log.Fatal(http.ListenAndServeTLS(":443", "configs/tls.crt", "configs/tls.key", nil))
    } else {
        log.Println("Listening without TLS on port 80")
        log.Fatal(http.ListenAndServe(":80", nil))
    }

}
