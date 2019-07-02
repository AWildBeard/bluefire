BUILD=go build
LDFLAGS=
GOOS=
GOARCH=
OUT=bluefire
SRC=*.go

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
	GOOS=${GOOS} GOARCH=${GOARCH} ${BUILD} ${LDFLAGS} -o ${OUT} ${SRC}

deploy:
	scp ${OUT} pi-2:

clean:
	rm -rf ${OUT}
