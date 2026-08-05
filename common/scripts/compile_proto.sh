#!/bin/bash
set -euo pipefail
protoc --go_out=. --go-grpc_out=. "proto/__PROTO_FILE__.proto"
