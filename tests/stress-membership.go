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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bbva/qed/api/apihttp"
)

type Config struct {
	maxGoRoutines  int
	numRequests    int
	apiKey         string
	startVersion   int
	continuous     bool
	balloonVersion uint64
	req            HTTPClient
}

type HTTPClient struct {
	client             *http.Client
	method             string
	endpoint           string
	expectedStatusCode int
}

// type Config map[string]interface{}

func NewDefaultConfig() *Config {
	return &Config{
		maxGoRoutines:  10,
		numRequests:    10000,
		apiKey:         "pepe",
		startVersion:   0,
		continuous:     false,
		balloonVersion: 9999,
		req: HTTPClient{
			client:             nil,
			method:             "POST",
			endpoint:           "http://localhost:8080",
			expectedStatusCode: 200,
		},
	}
}

type Task func(goRoutineId int, c *Config) ([]byte, error)

// func (t *Task) Timeout()

func SpawnerOfEvil(c *Config, t Task) {
	// TODO: only one client per run MAYBE
	var wg sync.WaitGroup

	for i := 0; i < c.maxGoRoutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Attacker(i, c, t)
		}(i)
	}
	wg.Wait()
}

// func BenchmarkMembership(b *testing.B) {
func Attacker(goRoutineId int, c *Config, f func(j int, c *Config) ([]byte, error)) {

	for j := c.startVersion + goRoutineId; j < c.startVersion+c.numRequests || c.continuous; j += c.maxGoRoutines {
		query, err := f(j, c)
		if len(query) == 0 {
			log.Fatalf("Empty query: %v", err)
		}

		req, err := http.NewRequest(c.req.method, c.req.endpoint, bytes.NewBuffer(query))
		if err != nil {
			log.Fatalf("Error preparing request: %v", err)
		}

		// Set Api-Key header
		req.Header.Set("Api-Key", c.apiKey)
		res, err := c.req.client.Do(req)
		defer res.Body.Close()
		if err != nil {
			log.Fatalf("Unable to perform request: %v", err)
		}
		if res.StatusCode != c.req.expectedStatusCode {
			log.Fatalf("Server error: %v", err)
		}

		io.Copy(ioutil.Discard, res.Body)
	}
}

func addSampleEvents(j int, c *Config) ([]byte, error) {
	buf := []byte(fmt.Sprintf("event %d", j))

	query, err := json.Marshal(
		&apihttp.Event{
			buf,
		},
	)

	return query, err
}

func queryMembership(j int, c *Config) ([]byte, error) {
	buf := []byte(fmt.Sprintf("event %d", j))

	query, err := json.Marshal(
		&apihttp.MembershipQuery{
			buf,
			c.balloonVersion,
		},
	)

	return query, err
}

func getVersion(eventTemplate string) uint64 {
	client := &http.Client{}

	buf := fmt.Sprintf(eventTemplate)

	query, err := json.Marshal(&apihttp.Event{[]byte(buf)})
	if len(query) == 0 {
		log.Fatalf("Empty query: %v", err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/events", bytes.NewBuffer(query))
	if err != nil {
		log.Fatalf("Error preparing request: %v", err)
	}

	// Set Api-Key header
	// TODO: remove pepe and pass a config var
	req.Header.Set("Api-Key", "pepe")
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		log.Fatalf("Unable to perform request: %v", err)
	}
	if res.StatusCode != 201 {
		log.Fatalf("Server error: %v", err)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var signedSnapshot apihttp.SignedSnapshot
	json.Unmarshal(body, &signedSnapshot)
	version := signedSnapshot.Snapshot.Version

	io.Copy(ioutil.Discard, res.Body)

	return version
}

func summary(op string, numRequestsf, elapsed float64, c *Config) {
	fmt.Printf(
		"%s done. Throughput: %.0f req/s: (%v reqs in %.3f seconds) | Concurrency: %d\n",
		op,
		numRequestsf/elapsed,
		c.numRequests,
		elapsed,
		c.maxGoRoutines,
	)
}

func stats(c *Config, t Task, message string) {
	numRequestsf := float64(c.numRequests)
	start := time.Now()
	SpawnerOfEvil(c, t)
	elapsed := time.Now().Sub(start).Seconds()
	summary(message, numRequestsf, elapsed, c)
}

func main() {
	fmt.Println("Starting contest...")

	client := &http.Client{}

	c := NewDefaultConfig()
	c.req.client = client
	c.req.expectedStatusCode = 201
	c.req.endpoint += "/events"

	numRequestsf := float64(c.numRequests)

	fmt.Println("Preloading events...")
	stats(c, addSampleEvents, "Preload")

	fmt.Println("Starting exclusive Query Membership...")
	cq := NewDefaultConfig()
	cq.req.client = client
	cq.req.expectedStatusCode = 200
	cq.req.endpoint += "/proofs/membership"
	stats(cq, queryMembership, "Query")

	fmt.Println("Starting continuous load...")
	ca := NewDefaultConfig()
	ca.req.client = client
	ca.req.expectedStatusCode = 201
	ca.req.endpoint += "/events"
	ca.startVersion = c.numRequests
	ca.continuous = true
	go SpawnerOfEvil(ca, addSampleEvents)

	fmt.Println("Starting Query Membership with continuous load...")
	//	stats(c, QueryMembership, "Read query")
	start := time.Now()
	SpawnerOfEvil(cq, queryMembership)
	elapsed := time.Now().Sub(start).Seconds()
	fmt.Printf(
		"Query done. Reading Throughput: %.0f req/s: (%v reqs in %.3f seconds) | Concurrency: %d\n",
		numRequestsf/elapsed,
		cq.numRequests,
		elapsed,
		cq.maxGoRoutines,
	)

	currentVersion := getVersion("last-event")
	fmt.Printf(
		"Query done. Writing Throughput: %.0f req/s: (%v reqs in %.3f seconds) | Concurrency: %d\n",
		(float64(currentVersion)-numRequestsf)/elapsed,
		currentVersion-uint64(c.numRequests),
		elapsed,
		c.maxGoRoutines,
	)
}