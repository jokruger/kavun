set dotenv-load

export PROJECTDIR := justfile_directory()
export BINARYCG := justfile_directory() + "/cmd/binary"

default:
    @just --list

generate:
    @go generate ./...

build: generate
    @mkdir -p ./build
    @go build -o ./build/kavun ./cmd/kavun

install: build
    @cp ./build/kavun $HOME/bin/

test: generate
    @go test -race ./tests/unit/fspec
    @go test -race ./tests/unit/parser
    @go test -race ./tests/unit/value
    @go test -race ./tests/unit/compiler
    @go test -race ./tests/unit/stdlib/json
    @go test -race ./tests/unit/stdlib
    @go test -race ./tests/unit
    @go run ./cmd/kavun -resolve ./tests/testdata/cli/test.kvn

bench-tool: generate
    @go run ./cmd/bench

clean:
    rm -rf ./build
    rm -rf ./dist
    rm -rf ./*.prof
    rm -rf ./*.log

bench-test: generate
    @go test -test.fullpath=true -run ^$ -bench=^BenchmarkVM$ -benchmem -cpuprofile cpu.prof -memprofile mem.prof -trace trace.prof ./tests/benchmark

cpu:
       go tool pprof -http=:8080 cpu.prof

mem:
    go tool pprof -http=:8080 mem.prof

trace:
       go tool trace trace.prof
