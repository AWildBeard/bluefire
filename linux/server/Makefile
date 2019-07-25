BUILD=go build
LDFLAGS=-X main.buildmode=debug
DATE=$(shell date '+%s')
GOOS=
GOARCH=
OUT=bluefire-server

.PHONY: release

all: help

help:
	@echo "Usage: make [\033[4marm-linux or amd64-linux\033[m]"

arm-linux: clean arm-linux-env
arm-linux-env:
	$(eval GOOS=linux)
	$(eval GOARCH=arm)

amd64-linux: clean amd64-linux-env
amd64-linux-env:
	$(eval GOOS=linux)
	$(eval GOARCH=amd64)

arm-linux amd64-linux:
	$(eval LDFLAGS=${LDFLAGS} -X main.buildversion=${DATE})
	GOOS=${GOOS} GOARCH=${GOARCH} ${BUILD} -ldflags="${LDFLAGS}" -o ${OUT}

release:
	$(eval LDFLAGS=-w -s -X main.buildmode=release)

deploy:
	scp ${OUT} pi-1:unstable

clean:
	rm -rf ${OUT}
