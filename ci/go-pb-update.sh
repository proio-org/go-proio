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
for proto in $(find proio -iname "*.proto"); do
    docker_run "protoc --gofast_out=tmp $proto"
done

# Move code to repo
docker_run "mv tmp/proio/* go-proio-pb/"

# Initialize go module
cd go-proio-pb
GO111MODULE=on go mod init github.com/proio-org/go-proio-pb
GO111MODULE=on go get -u ./...

# Create and push commit
git add --all
git commit -m "Automatic generated code update via Travis CI" -m "go-proio commit: $TRAVIS_COMMIT"
git push "https://$GO_PB_REPO_TOKEN@$GO_PB_REPO_REF"

exit 0
