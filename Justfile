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
    @go test -race -timeout 5s ./...
    @go run ./cmd/kavun -resolve ./testdata/cli/test.kvn

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
