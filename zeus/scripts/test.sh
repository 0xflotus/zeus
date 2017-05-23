#!/bin/bash

# {zeus}
# dependencies: reset
# description: start tests
# help: zeus test script
# {zeus}

echo "starting tests"

go test -v -coverprofile coverage.out -cover

if [[ $? == 0 ]]; then
	go tool cover -html=coverage.out
fi
