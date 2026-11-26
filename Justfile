set dotenv-load

export PROJECTDIR := justfile_directory()
export BINARYCG := justfile_directory() + "/cmd/binary"

default:
    @just --list

# the -docs-url of gen.go's //go:generate line: the docs field of kavun_reference.json
docs_url := "https://github.com/jokruger/kavun/blob/main/docs"

# gen.go's two //go:generate lines: kavun-gen (every kavun_gen.go + kavun_reference.json), then kavun-docs
# (docs/reference/ rendered from kavun_reference.json)
generate:
    @go generate ./...

# the same two stages, comparing instead of writing (clean = in sync)
generate-check:
    @go run ./cmd/kavun-gen -check -docs-url {{docs_url}} ./...
    @go run ./cmd/kavun-docs -check -out docs/reference kavun_reference.json

build: generate
    @mkdir -p ./build
    @go build -o ./build/kavun ./cmd/kavun

install: build
    @cp ./build/kavun $HOME/bin/

test: generate
    @go test -race -timeout 60s ./...
    @go run ./cmd/kavun -resolve ./testdata/cli/test.kvn

# the harness scripts again at an optimization level (default O3): the optimizer must not change a result
test-opt level="O3": generate
    @KAVUN_TEST_OPT={{level}} go test -race -timeout 120s .

bench-tool: generate
    @go run ./cmd/bench

bench-accept:
    cp ./bench-current.json ./bench-baseline.json

clean:
    rm -rf ./build
    rm -rf ./dist
    rm -rf ./*.prof
    rm -rf ./*.log

bench-test: generate
    @go test -test.fullpath=true -run ^$ -bench=^BenchmarkVM$ -benchmem -cpuprofile cpu.prof -memprofile mem.prof -trace trace.prof ./benchmarks

cpu:
       go tool pprof -http=:8080 cpu.prof

mem:
    go tool pprof -http=:8080 mem.prof

trace:
       go tool trace trace.prof
