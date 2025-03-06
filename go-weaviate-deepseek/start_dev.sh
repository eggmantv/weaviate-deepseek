#!/bin/bash
_IS_CHILD=1 \
    REDIS_URI=redis://localhost:6379/1 \
    HOST_IP=localhost \
    go run cmd/main.go -e development
