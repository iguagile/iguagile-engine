#!/usr/bin/env bash

cnt=0

export REDIS_HOST=localhost:6379

for i in 1 2 3 4 5 6 7 8 9 10
do
    echo "testing try... $i"
    go test -v ./iguagile -count=1
    res=$?
    if [[ ${res} -ne 0 ]]; then
        echo "test failed."
        exit 1;
    fi

done

