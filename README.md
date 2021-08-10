# Nest Smart Device Management (and video capture!)

This repository implements a Go client for the [Nest Smart Device Management
API](https://developers.google.com/nest/device-access/reference/rest/) for
Cameras, as well as some tools for capturing video streams using those APIs.

## Device Access Setup

* Create (or choose an existing Cloud project).
* Enable the [Smart Device Management
  API](https://console.cloud.google.com/apis/library/smartdevicemanagement.googleapis.com)
* [Create an OAuth
  Token](https://console.cloud.google.com/apis/api/smartdevicemanagement.googleapis.com/credentials).
  Be sure to create a "Desktop App" token type.
* Go to the [Device Access Console](https://console.nest.google.com/device-access/project-list).  You may need to register as a developer.
* Create a project.  (You'll need the Project ID later to find your Device IDs.)
  Use the OAuth Client ID you created above.
* Use [Partner Connection Manager](https://nestservices.google.com/partnerconnections)
  to connect devices to the project.  Enable the `Allow PROJECTNAME to see and
  display your camera’s livestream` permission for the devices you want to
  interact with.

## Use Cases

### Common setup

Use cmd/list/list.go to get the device id.

    go run ./cmd/list --projectid=$PROJECTID --oauth_clientid=... --oauth_secret=...

This will also create a `token.json` in the current directory (if it doesn't exist) with authentication tokens.

Set the `DEVICE` environment variable to your Device ID.

    DEVICE=enterprises/81b04b74-.../devices/...

### Quick Capture

Capture 15 seconds into `/tmp/15s.mp4'

    go run ./cmd/capture --device=$DEVICE \
      --once --file_spec=/tmp/15s.mp4 --duration 15s

### Long Capture

Capture until cancelled, writing to /tmp/B-YYYYMMDD-HHMMSS.mp4, rotating new
files every 5 minutes.

    go run cmd/capture/capture.go --device=$DEVICE \
      --oauth_clientid=... --oauth_secret=... \
      --file_spec=/tmp/B-%Y%m%d-%H%M%S.mp4 \
      --segment_time=5m

Doing this with short segments makes it easy to process chunks later without
using a video editor. as you can just ignore/remove the files you're not
interested in.

#### Time Lapse

Here's how to create a time lapse video from the segments recorded above.

    cd /tmp/
    # create list of all segments
    rm segments
    for f in B-*.mp4; do echo "file '$f'" >> segments; done
    # concatanate segments into one big mp4 files.
    ffmpeg -f concat -safe 0 -i segments  -c copy all.mp4
    # create time lapse at 24x speed without audio
    ffmpeg -i all.mp4 -filter:v "setpts=0.041*PTS" -an timelapse.041.mp4

To compute the `setpts` value, use `1/multiplier` .  (i.e. `10x` speedup would be `1/10` => `0.1` )

`-an` drops audio.  

Alternatively, you can skip the explicit concatenation step and generate the time lapse from multiple files with one ffmpeg command:

    ffmpeg -f concat -safe 0 -i segments -filter:v "setpts=0.041*PTS" -an timelapse.mp4

### Stream To YouTube

If your YouTube channel (or Twitch.tv or similar) supports livestreaming, you can...

    go run cmd/capture/capture.go --device=$DEVICE \
      --oauth_clientid=... --oauth_secret=... \
      --stream_to=rtmps://a.rtmp.youtube.com/live2/$STRTEAMKEY

### Config Options

Passing the OAuth information every time is annoying.

You can create a file that looks like this:

    oauth_clientid=123456787-...something.apps.googleusercontent.com
    oauth_secret=Xaqs05W_k9K-...
    projectid=enterprises/81b04b74-...

And then pass it with the `--config` argument to any of the tools.

(This functionality brought to you by <https://github.com/namsral/flag>.  You can also use environment variables.)

## More Links

* [Nest Device Access](https://developers.google.com/nest/device-access)
* [Partner Connection Manager](https://nestservices.google.com/partnerconnections)
* [API Rate Limits](https://developers.google.com/nest/device-access/project/limits)

## Disclaimer

This is not an official Google project.
