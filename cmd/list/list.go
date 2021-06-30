// lists all video capable devices exposed in the Nest Smart Device Management API for the specified project.
package main

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
	"time"

	"github.com/namsral/flag"

	"github.com/rspier/nestsdm"
	"google.golang.org/api/option"
	"google.golang.org/api/smartdevicemanagement/v1"
)

var (
	projectID     = flag.String("projectid", "", "project ID from https://console.nest.google.com/device-access/project-list")
	oAuthClientID = flag.String("oauth_clientid", "", "OAuth2 Client ID")
	oAuthSecret   = flag.String("oauth_secret", "", "OAuth2 Secret ID")
	tokFile       = flag.String("token_file", "token.json", "path to oauth token cache")
)

func main() {
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.Parse()

	c := nestsdm.OAuthClient(*oAuthClientID, *oAuthSecret, *tokFile)
	sdm, err := smartdevicemanagement.NewService(context.Background(), option.WithHTTPClient(c))
	if err != nil {
		log.Fatalf("smartdevicemanagement.NewService(): %v", err)
	}

	ctx, cxl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cxl()

	l := sdm.Enterprises.Devices.List(*projectID)
	devs, err := l.Context(ctx).Do()
	if err != nil {
		log.Fatalf("sdm.Enterprise.Devices.List(%q): %v", *projectID, err)
	}

	for _, d := range devs.Devices {
		j, _ := d.Traits.MarshalJSON()
		if err != nil {
			log.Fatalf("can't marshal json: %v", err)
		}

		var ds nestsdm.Traits
		err = json.Unmarshal(j, &ds)
		if err != nil {
			log.Fatalf("can't unmarshal json: %v", err)
		}

		// No codecs supported?  It's not a camera.  TODO: Just check for the CameraLiveStream trait.
		if len(ds.CameraLiveStream.VideoCodecs) == 0 {
			continue
		}

		room := ""
		for _, p := range d.ParentRelations {
			if p.DisplayName != "" {
				room = p.DisplayName
				break
			}
		}

		cn := ""
		if ds.Info.CustomName != "" {
			cn = "(" + ds.Info.CustomName + ")"
		}

		fmt.Printf("%s%s: %s\n", room, cn, d.Name)
	}
}
