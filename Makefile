# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ggda android ios ggda-cross swarm evm all test clean
.PHONY: ggda-linux ggda-linux-386 ggda-linux-amd64 ggda-linux-mips64 ggda-linux-mips64le
.PHONY: ggda-linux-arm ggda-linux-arm-5 ggda-linux-arm-6 ggda-linux-arm-7 ggda-linux-arm64
.PHONY: ggda-darwin ggda-darwin-386 ggda-darwin-amd64
.PHONY: ggda-windows ggda-windows-386 ggda-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

ggda:
	build/env.sh go run build/ci.go install ./cmd/ggda
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ggda\" to launch ggda."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/ggda.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Ggda.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

ggda-cross: ggda-linux ggda-darwin ggda-windows ggda-android ggda-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/ggda-*

ggda-linux: ggda-linux-386 ggda-linux-amd64 ggda-linux-arm ggda-linux-mips64 ggda-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-*

ggda-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/ggda
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep 386

ggda-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ggda
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep amd64

ggda-linux-arm: ggda-linux-arm-5 ggda-linux-arm-6 ggda-linux-arm-7 ggda-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep arm

ggda-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ggda
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep arm-5

ggda-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ggda
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep arm-6

ggda-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ggda
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep arm-7

ggda-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ggda
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep arm64

ggda-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/ggda
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep mips

ggda-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/ggda
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep mipsle

ggda-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ggda
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep mips64

ggda-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ggda
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ggda-linux-* | grep mips64le

ggda-darwin: ggda-darwin-386 ggda-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ggda-darwin-*

ggda-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ggda
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-darwin-* | grep 386

ggda-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ggda
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-darwin-* | grep amd64

ggda-windows: ggda-windows-386 ggda-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ggda-windows-*

ggda-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ggda
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-windows-* | grep 386

ggda-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ggda
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggda-windows-* | grep amd64
