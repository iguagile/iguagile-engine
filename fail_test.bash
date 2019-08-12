#!/usr/bin/env bash

cnt=0

export REDIS_HOST=localhost:6379
go test -v ./iguagile -count=1

res=$?
cnt=$((cnt++))

if [ $res -ne 0 ] || [ $cnt -ge 10 ]; then exit 1; fi

