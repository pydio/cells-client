DEV_VERSION=5.0.0-dev
ENV=env GOOS=linux
TIMESTAMP:=$(shell date -u +%Y%m%d%H%M%S)
CELLS_VERSION?=${DEV_VERSION}.${TIMESTAMP}
MOD_UPDATE?=v5-dev

.PHONY: all clean main linux arm arm64 win darwin xgo

linux: linux-amd64

arm64: linux-arm64

arm: linux-arm

win: windows-amd64

darwin: darwin-arm64

linux-amd64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

linux-arm:
	env CGO_ENABLED=0 GOOS=linux GOARM=7 GOARCH=arm go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

linux-arm64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

windows-amd64:
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec.exe .

darwin-arm64:
	env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

darwin-amd64:
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -trimpath \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

# The 2 below targets build a binary that is adapted to the OS / Arch that runs the build.
main:
	env CGO_ENABLED=0 go build -a -trimpath\
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${CELLS_VERSION}" \
	 -o cec .

dev:
	env CGO_ENABLED=0 go build -tags dev \
	 -ldflags "-X github.com/pydio/cells-client/v5/common.Version=${DEV_VERSION}"\
	 -o cec\
	 .

## NOTE: we expect that the SDK and Cells client projects are in the same folder.
mod-local:
	go mod edit -replace github.com/pydio/cells-sdk-go/v5=../cells-sdk-go

mod-update:
	go mod edit -dropreplace github.com/pydio/cells-sdk-go/v5
	go get -d github.com/pydio/cells-sdk-go/v5@${MOD_UPDATE}
	go mod download github.com/pydio/cells-sdk-go/v5
	GONOSUMDB=* go mod tidy

clean:
	rm -f cec
