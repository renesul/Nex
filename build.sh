#!/bin/bash
pkill -f ./nex 2>/dev/null && sleep 0.5
CGO_ENABLED=1 go build -tags "fts5" -o nex ./cmd/nex/
