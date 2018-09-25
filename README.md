# proio for Go
[![Build Status](https://travis-ci.org/proio-org/go-proio.svg?branch=master)](https://travis-ci.org/proio-org/go-proio)
[![codecov](https://codecov.io/gh/proio-org/go-proio/branch/master/graph/badge.svg)](https://codecov.io/gh/proio-org/go-proio)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/706bdf4f827c4bb7adbae4bdc6b07662)](https://www.codacy.com/app/proio-org/go-proio?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=proio-org/go-proio&amp;utm_campaign=Badge_Grade)

Please see the [main proio repository](https://github.com/proio-org/proio) for general information on proio.

## API
API documentation is provided by godoc.org

[![GoDoc](https://godoc.org/github.com/proio-org/go-proio?status.svg)](https://godoc.org/github.com/proio-org/go-proio)

## Installation
go-proio and included [command-line tools](tools) are `go get`-able.  Make sure
you have the `go` compiler installed and set up:
```shell
go get github.com/proio-org/go-proio/...
```

If you do not have the `go` compiler, you can find pre-compiled binaries for
the tools [in the releases](https://github.com/proio-org/go-proio/releases).

For information on what versions of Go are supported, please see the [Travis CI
page](https://travis-ci.org/proio-org/go-proio).

## Examples
* [Print](example_print_test.go)
* [Scan](example_scan_test.go)
* [Skip](example_skip_test.go)
* [Push, get, inspect](example_push_get_inspect_test.go)
