DEV_VERSION=4.0.0-dev
ENV=env GOOS=linux
TIMESTAMP:=$(shell date -u +%Y%m%d%H%M%S)
CELLS_CLIENT_VERSION?=${DEV_VERSION}.${TIMESTAMP}

XGO_TARGETS?="linux/amd64,darwin/amd64,windows/amd64"
XGO_IMAGE?=techknowlogick/xgo:go-1.21.x
XGO_BIN?=${GOPATH}/bin/xgo

.PHONY: all clean main dev xgo

main:
	export CGO_ENABLED=0
	go build -a\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec\
	 .

linux:
	export CGO_ENABLED=0
	env GOOS=linux GOARCH=amd64 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec .

arm:
	export CGO_ENABLED=0
	env GOOS=linux GOARM=7 GOARCH=arm go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec .

arm64:
	export CGO_ENABLED=0
	env GOOS=linux GOARCH=arm64 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec .

win:
	export CGO_ENABLED=0
	env GOOS=windows GOARCH=amd64 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec.exe .

mac:
	export CGO_ENABLED=0
	env GOOS=darwin GOARCH=amd64 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -o cec .

xgo:
	${XGO_BIN} -go 1.21 \
	 --image ${XGO_IMAGE} \
	 --targets ${XGO_TARGETS} \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}"\
	 -out cec\
	 .

dev:
	export CGO_ENABLED=0
	go build \
	 -tags dev \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${DEV_VERSION}"\
	 -o cec\
	 .

clean:
	rm -f cec
