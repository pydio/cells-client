DEV_VERSION=4.3.1-dev
ENV=env GOOS=linux
TIMESTAMP:=$(shell date -u +%Y%m%d%H%M%S)
CELLS_CLIENT_VERSION?=${DEV_VERSION}.${TIMESTAMP}
MOD_UPDATE?=v5-dev

.PHONY: all clean main linux arm arm64 win darwin xgo

main:
	env CGO_ENABLED=0 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec .

linux:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec .

arm:
	env CGO_ENABLED=0 GOOS=linux GOARM=7 GOARCH=arm go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec .

arm64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec .

win:
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec.exe .

darwin:
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${CELLS_CLIENT_VERSION}" \
	 -o cec .

dev:
	env CGO_ENABLED=0 go build -tags dev \
	 -ldflags "-X github.com/pydio/cells-client/v4/common.Version=${DEV_VERSION}"\
	 -o cec\
	 .

## We assume the sdk and cells client projects are in the same folder...
mod-local:
	go mod edit -replace github.com/pydio/cells-sdk-go/v4=../cells-sdk-go

mod-update:
	go mod edit -dropreplace github.com/pydio/cells-sdk-go/v4
	go get -d github.com/pydio/cells-sdk-go/v4@${MOD_UPDATE}
	go mod download github.com/pydio/cells-sdk-go/v4
	GONOSUMDB=* go mod tidy

clean:
	rm -f cec
