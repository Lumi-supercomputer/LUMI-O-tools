
GH=$(shell git rev-parse --short HEAD)
V=$(shell git tag --points-at HEAD)
T=$(shell date +"%Y-%m-%dT%H:%M:%S%z")


all: lumio-conf

lumio-conf: cmd/lumio-conf/main.go internal/toolConfig/*.go internal/util/*.go
	go build -ldflags="-X 'lumioconf/internal/util.buildTime=$(T)' -X lumioconf/internal/util.gitHash=$(GH) -X lumioconf/internal/util.progVersion=$(V)" ./cmd/lumio-conf/

.PHONY: clean
clean: 
	rm -f lumio-conf
