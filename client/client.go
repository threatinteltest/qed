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

// Package client implements the command line interface to interact with
// the REST API
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bbva/qed/balloon"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/protocol"
)

// HTTPClient ist the stuct that has the required information for the cli.
type HTTPClient struct {
	conf *Config

	*http.Client
}

// NewHTTPClient will return a new instance of HTTPClient.
func NewHTTPClient(conf Config) *HTTPClient {
	var tlsConf *tls.Config

	if conf.Insecure {
		tlsConf = &tls.Config{InsecureSkipVerify: true}
	} else {
		tlsConf = &tls.Config{}
	}

	return &HTTPClient{
		&conf,
		&http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
				TLSClientConfig:     tlsConf,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}

}

func (c HTTPClient) exponentialBackoff(req *http.Request) (*http.Response, error) {

	var retries uint

	for {
		resp, err := c.Do(req)
		if err != nil {
			if retries == 5 {
				return nil, err
			}
			retries = retries + 1
			delay := time.Duration(10 << retries * time.Millisecond)
			time.Sleep(delay)
			continue
		}
		return resp, err
	}

}

func (c HTTPClient) doReq(method, path string, data []byte) ([]byte, error) {
	url, err := url.Parse(c.conf.Endpoint + path)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s", url), bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.conf.APIKey)

	resp, err := c.exponentialBackoff(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("Unexpected server error")
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, fmt.Errorf("Invalid request")
	}

	return bodyBytes, nil

}

// Add will do a request to the server with a post data to store a new event.
func (c HTTPClient) Add(event string) (*protocol.Snapshot, error) {

	data, _ := json.Marshal(&protocol.Event{[]byte(event)})

	body, err := c.doReq("POST", "/events", data)
	if err != nil {
		return nil, err
	}

	var snapshot protocol.Snapshot
	json.Unmarshal(body, &snapshot)

	return &snapshot, nil

}

// Membership will ask for a Proof to the server.
func (c HTTPClient) Membership(key []byte, version uint64) (*protocol.MembershipResult, error) {

	query, _ := json.Marshal(&protocol.MembershipQuery{
		key,
		version,
	})

	body, err := c.doReq("POST", "/proofs/membership", query)
	if err != nil {
		return nil, err
	}

	var proof *protocol.MembershipResult
	json.Unmarshal(body, &proof)

	return proof, nil

}

// Membership will ask for a Proof to the server.
func (c HTTPClient) MembershipDigest(keyDigest hashing.Digest, version uint64) (*protocol.MembershipResult, error) {

	query, _ := json.Marshal(&protocol.MembershipDigest{
		keyDigest,
		version,
	})

	body, err := c.doReq("POST", "/proofs/digest-membership", query)
	if err != nil {
		return nil, err
	}

	var proof *protocol.MembershipResult
	json.Unmarshal(body, &proof)

	return proof, nil

}

// Incremental will ask for an IncrementalProof to the server.
func (c HTTPClient) Incremental(start, end uint64) (*protocol.IncrementalResponse, error) {

	query, _ := json.Marshal(&protocol.IncrementalRequest{
		start,
		end,
	})

	body, err := c.doReq("POST", "/proofs/incremental", query)
	if err != nil {
		return nil, err
	}

	var response *protocol.IncrementalResponse
	json.Unmarshal(body, &response)

	return response, nil
}

func uint2bytes(i uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, i)
	return bytes
}

// Verify will compute the Proof given in Membership and the snapshot from the
// add and returns a proof of existence.
func (c HTTPClient) Verify(
	result *protocol.MembershipResult,
	snap *protocol.Snapshot,
	hasherF func() hashing.Hasher,
) bool {

	proof := protocol.ToBalloonProof(result, hasherF)

	return proof.Verify(snap.EventDigest, &balloon.Snapshot{
		snap.EventDigest,
		snap.HistoryDigest,
		snap.HyperDigest,
		snap.Version,
	})

}

// Verify will compute the Proof given in Membership and the snapshot from the
// add and returns a proof of existence.
func (c HTTPClient) DigestVerify(
	result *protocol.MembershipResult,
	snap *protocol.Snapshot,
	hasherF func() hashing.Hasher,
) bool {

	proof := protocol.ToBalloonProof(result, hasherF)

	return proof.DigestVerify(snap.EventDigest, &balloon.Snapshot{
		snap.EventDigest,
		snap.HistoryDigest,
		snap.HyperDigest,
		snap.Version,
	})

}

func (c HTTPClient) VerifyIncremental(
	result *protocol.IncrementalResponse,
	startSnapshot, endSnapshot *protocol.Snapshot,
	hasher hashing.Hasher,
) bool {

	proof := protocol.ToIncrementalProof(result, hasher)

	start := &balloon.Snapshot{
		startSnapshot.EventDigest,
		startSnapshot.HistoryDigest,
		startSnapshot.HyperDigest,
		startSnapshot.Version,
	}
	end := &balloon.Snapshot{
		endSnapshot.EventDigest,
		endSnapshot.HistoryDigest,
		endSnapshot.HyperDigest,
		endSnapshot.Version,
	}

	return proof.Verify(start, end)

}
