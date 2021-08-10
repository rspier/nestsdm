#!/bin/busybox sh

# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# 	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [[ -z "$DEVICE" ]]; then
    echo missing required parameter DEVICE.
    fail=1
fi
if [[ -z "$OAUTH_CLIENTID" ]]; then
    echo missing required parameter OAUTH_CLIENTID.
    fail=1
fi
if [[ -z "$OAUTH_SECRET" ]]; then
    echo missing required parameter OAUTH_SECRET.
    fail=1
fi
if [[ "${fail}" == "1" ]]; then
    exit 1
fi

FILE_SPEC="${FILE_SPEC:-/tmp/B-%Y%m%d-%H%M%S.mp4}"
SEGMENT_TIME="${SEGMENT_TIME:-5m}"
TOKEN_FILE="${TOKEN_FILE:-token.json}"

/usr/local/bin/capture \
    --device="${DEVICE}" \
    --oauth_clientid="${OAUTH_CLIENTID}" \
    --oauth_secret="${OAUTH_SECRET}" \
    --file_spec="${FILE_SPEC}" \
    --segment_time=${SEGMENT_TIME}