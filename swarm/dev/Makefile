.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/gdachain/go-gdachain/cmd/swarm
	go build -o bin/ggda     github.com/gdachain/go-gdachain/cmd/ggda
	go build -o bin/bootnode github.com/gdachain/go-gdachain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/gdachain/go-gdachain/swarm/...
