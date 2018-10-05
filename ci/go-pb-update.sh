#!/bin/bash

git clone https://git@$GO_PB_REPO_REF go-proio-pb

mkdir tmp

GOFAST=$(which protoc-gen-gofast)
docker_run () {
    docker run \
        -v $TRAVIS_BUILD_DIR:$TRAVIS_BUILD_DIR \
        -v $GOFAST:$GOFAST \
        -w $PWD \
        $DOCKER_AUX_REPO \
        bash -c 'PATH='$(dirname $GOFAST)':$PATH '"$1"
}

# Generate protobuf message code
rm -rf go-proio-pb/*
for proto in $(find proto -iname "*.proto"); do
    if [ -z "$(grep -i go_package $proto)" ]; then
        go_package=$(basename ${proto%.proto})
        if [ "$go_package" == "proio" ]; then
            go_package=proto
        fi
        echo "option go_package = \"$go_package\";" >> $proto
    fi
    docker_run "protoc \
        --proto_path=proio/model=proto/model \
        --proto_path=proio/proto=proto \
        --gofast_out=tmp $proto"
done

# Move code to repo
docker_run "mv tmp/proio/proto/* go-proio-pb/"
docker_run "mv tmp/proio/model go-proio-pb/"

# Initialize go module
cd go-proio-pb
GO111MODULE=on go mod init github.com/proio-org/go-proio-pb
GO111MODULE=on go get -u ./...

# Create and push commit
git add --all
git commit -m "Automatic generated code update via Travis CI" -m "go-proio commit: $TRAVIS_COMMIT"
git push "https://$GO_PB_REPO_TOKEN@$GO_PB_REPO_REF"

exit 0
