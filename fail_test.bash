#!/usr/bin/env bash

cnt=0

export REDIS_HOST=localhost:6379
go test -v ./iguagile -count=1 -o iguagile-test

for i in {1..10}
do
    echo "testing try... $i"
    ./iguagile-test
    res=$?
    if [[ ${res} -ne 0 ]]; then
        echo "test failed."
        exit 1;
    fi

done

