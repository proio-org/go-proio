#!/bin/bash

mkdir -p go-proio-pb

rm -rf tmp
mkdir tmp

# Generate protobuf message code
rm -rf go-proio-pb/*
for proto in $(find proio -iname "*.proto"); do
    if [ -z "$(grep -i go_package $proto)" ]; then
        go_package=$(basename ${proto%.proto})
        if [ "$go_package" == "proio" ]; then
            go_package=proto
        fi
        echo "option go_package = \"$go_package\";" >> $proto
    fi
    protoc --gofast_out=tmp $proto
done

# Move code to repo
mv tmp/proio/* go-proio-pb/

# Initialize go module
cd go-proio-pb
GO111MODULE=on go mod init github.com/proio-org/go-proio-pb
GO111MODULE=on go get -u ./...

exit 0
