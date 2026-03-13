set dotenv-load

export PROJECTDIR := justfile_directory()
export BINARYCG := justfile_directory() + "/cmd/binary"

default:
    @just --list

generate:
    @go generate ./...

build: generate
    @mkdir -p ./build
    @go build -o ./build/gs ./cmd/gs

test: generate
    @go test -race -cover ./...
    @go run ./cmd/gs -resolve ./testdata/cli/test.gs

clean:
    rm -rf ./build
    rm -rf ./*.prof
    rm -rf ./*.log
