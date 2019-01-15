#!/bin/bash

if [ "$GO_PB_REPO_REF" == "" ]; then
    GO_PB_REPO_REF="github.com/proio-org/go-proio-pb"
fi

pbdir=$PWD/go-proio-pb
mkdir -p $pbdir
rm -rf $pbdir/*
tmpdir=$(mktemp -d)

# Generate protobuf message code
for proto in $(find proto -iname "*.proto"); do
    if [ -z "$(grep -i go_package $proto)" ]; then
        go_package=$(basename ${proto%.proto})
        if [ "$go_package" == "proio" ]; then
            go_package=proto
        fi
        echo "option go_package = \"$go_package\";" >> $proto
    fi
    protoc \
        --proto_path=proio/model=proto/model \
        --proto_path=proio/proto=proto \
        --gofast_out=$tmpdir $proto
done

# Move code to repo
mv $tmpdir/proio/proto/* $pbdir/
mv $tmpdir/proio/model $pbdir/
rm -rf $tmpdir

# Initialize go module
cd $pbdir
GO111MODULE=on go mod init $GO_PB_REPO_REF
GO111MODULE=on go get -u ./...

exit 0
