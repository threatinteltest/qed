/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

func main() {
	conn := flag.Int("c", 1, "Max open idle connections")
	workers := flag.Uint64("w", 1, "Request per second")
	timeout := flag.Duration("t", 30*time.Second, "Timeout")
	duration := flag.Duration("d", 10*time.Second, "Duration")
	endpoint := flag.String("e", "http://localhost:8080/events", "Endpoint")
	apikey := flag.String("k", "apikey", "apikey")
	rate := flag.Int("r", 100, "Request per second")
	flag.Parse()

	targeter := myTargeter(*endpoint, http.Header{"Api-Key": []string{*apikey}})

	atk := vegeta.NewAttacker(vegeta.Connections(*conn), vegeta.Workers(*workers), vegeta.Timeout(*timeout))
	vgrate := vegeta.Rate{Freq: *rate, Per: time.Second}
	res := atk.Attack(targeter, vgrate, *duration, "attack")
	enc := vegeta.NewEncoder(os.Stdout)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			atk.Stop()
			os.Exit(0)
		case r, ok := <-res:
			if !ok {
				os.Exit(-1)
			}
			if err := enc.Encode(r); err != nil {
				os.Exit(-1)
			}
		}
	}

}

func myTargeter(endpoint string, hdr http.Header) vegeta.Targeter {
	var mu sync.Mutex

	return func(tgt *vegeta.Target) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		event := base64.StdEncoding.EncodeToString([]byte(time.Now().String()))

		tgt.Body = []byte(fmt.Sprintf(`{"Event": "%s"}`, event))
		tgt.Header = hdr
		tgt.Method = "POST"
		tgt.URL = endpoint
		return nil
	}
}
