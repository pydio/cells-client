DEV_VERSION=4.0.0-dev
ENV=env GOOS=linux
TIMESTAMP:=$(shell date -u +%Y%m%d%H%M%S)
CELLS_CLIENT_VERSION?=${DEV_VERSION}.${TIMESTAMP}

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

clean:
	rm -f cec
