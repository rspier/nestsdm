// Package nestsdm implements a client for the Nest Smart Device Management APIs (for cameras)
package nestsdm

/*

Copyright 2021 Google LLC

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

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/smartdevicemanagement/v1"
)

const ()

// The SDM Go API doesn't actually include common types/structs for the return values.  So we implement them here:

type InfoTrait struct {
	CustomName string `json:"customName"`
}

type VideoResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type CameraLiveStreamTrait struct {
	MaxVideoResolution VideoResolution `json:"maxVideoResolution"`
	VideoCodecs        []string        `json:"videoCodecs"`
	AudioCodecs        []string        `json:"audioCodecs"`
}
type CameraImageTrait struct {
	MaxVideoResolution VideoResolution `json:"maxVideoResolution"`
}

type Traits struct {
	Info             InfoTrait              `json:"sdm.devices.traits.Info"`
	CameraLiveStream *CameraLiveStreamTrait `json:"sdm.devices.traits.CameraLiveStream"`
	CameraImage      *CameraImageTrait      `json:"sdm.devices.traits.CameraImage"`
}

type ResultStreamURLs struct {
	RTSPURL string `json:"rtspUrl"`
}
type GenerateRtspStreamResults struct {
	StreamURLs           ResultStreamURLs `json:"streamUrls"`
	StreamToken          string           `json:"streamToken"`
	StreamExtensionToken string           `json:"streamExtensionToken"`
	ExpiresAt            time.Time        `json:"expiresAt"`
}

type ExtendRtspStreamResults struct {
	StreamToken          string    `json:"streamToken"`
	StreamExtensionToken string    `json:"streamExtensionToken"`
	ExpiresAt            time.Time `json:"expiresAt"`
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokFile string) *http.Client {
	// The tokFile stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	authURL += "&prompt=consent"

	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func OAuthClient(clientID, secret, tokFile string) *http.Client {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		Scopes:       []string{"https://www.googleapis.com/auth/sdm.service"},
		Endpoint:     google.Endpoint,
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
	}
	return getClient(cfg, tokFile)
}

func extend(ctx context.Context, sdm *smartdevicemanagement.Service, set string, device string) (time.Time, string) {

	bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	var res *ExtendRtspStreamResults
	err := backoff.Retry(func() error {
		var err error
		log.Print("extending stream")
		res, err = ExtendRTSPStream(ctx, sdm, device, set)
		return err
	}, bo)

	if err != nil {
		log.Printf("ERROR extendRTSPStream: %v", err)
		return time.Time{}, ""
	}

	log.Printf("expiration extended to %v", res.ExpiresAt)

	return res.ExpiresAt, res.StreamExtensionToken
}

func Extender(ctx context.Context, sdm *smartdevicemanagement.Service, r *GenerateRtspStreamResults, device string) {
	expire := r.ExpiresAt
	set := r.StreamExtensionToken

	log.Printf("expiration started at %v", r.ExpiresAt)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// try and extend 1 minute before it expires
		if time.Now().After(expire.Add(-1 * time.Minute)) {
			ctx, cxl := context.WithDeadline(ctx, expire)
			e, s := extend(ctx, sdm, set, device)
			defer cxl()
			// successful renew
			if e.After(expire) {
				expire = e
				set = s
			}
		}
		time.Sleep(time.Second)
	}
}

func ExtendRTSPStream(ctx context.Context, sdm *smartdevicemanagement.Service, device, set string) (*ExtendRtspStreamResults, error) {

	p := struct {
		StreamExtensionToken string `json:"streamExtensionToken"`
	}{
		StreamExtensionToken: set,
	}
	pj, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %v", err)
	}
	rj := googleapi.RawMessage(pj)

	req := &smartdevicemanagement.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.CameraLiveStream.ExtendRtspStream",
		Params:  rj,
	}

	c := sdm.Enterprises.Devices.ExecuteCommand(device, req)
	got, err := c.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("calling ExecuteCommand: %v", err)
	}

	j, err := got.Results.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("got.Results.MarshalJSON(): %v", err)
	}

	var res ExtendRtspStreamResults
	err = json.Unmarshal(j, &res)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %v", err)
	}

	return &res, nil
}

func GenerateRTSPStream(ctx context.Context, sdm *smartdevicemanagement.Service, device string) (*GenerateRtspStreamResults, error) {
	req := &smartdevicemanagement.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.CameraLiveStream.GenerateRtspStream",
	}
	c := sdm.Enterprises.Devices.ExecuteCommand(device, req)
	got, err := c.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("calling ExecuteCommand: %v", err)
	}

	j, err := got.Results.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("got.Results.MarshalJSON(): %v", err)
	}

	var res GenerateRtspStreamResults
	err = json.Unmarshal(j, &res)

	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %v", err)
	}

	return &res, nil
}
