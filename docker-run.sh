#!/bin/sh

# Sample of how to run the nestsdm container.  Replace [[PLACEHOLDER]]s before
# use.

# Maps the token file in the current directory into the container. 

docker run --rm -ti \
    -e DEVICE=enterprises/[[ENTERPRISEID]]/devices/[[DEVICEID]] \
    -e OAUTH_CLIENTID=[[CLIENTID]].apps.googleusercontent.com \
    -e OAUTH_SECRET=[[SECRET]] \
    -e TOKEN_FILE=/tmp/token.json \
    -e FILE_SPEC="/out/A-%Y%m%d-%H%M%S.mp4" \
    -e SEGMENT_TIME="15m" \
    -v $PWD/token.json:/tmp/token.json \
    -v /tmp/out:/out \
    nestsdm:latest