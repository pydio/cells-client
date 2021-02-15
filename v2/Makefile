DEV_VERSION=2.1.0-dev
ENV=env GOOS=linux
TODAY:=$(shell date -u +%Y-%m-%dT%H:%M:%S)
TIMESTAMP:=$(shell date -u +%Y%m%d%H%M%S)
GITREV:=$(shell git rev-parse HEAD)
CELLS_CLIENT_VERSION?=${DEV_VERSION}.${TIMESTAMP}
XGO_TARGETS?="linux/amd64,darwin/amd64,windows/amd64"


main:
	go build -a\
	 -ldflags "-X github.com/pydio/cells-client/v2/common.Version=${CELLS_CLIENT_VERSION}\
	 -X github.com/pydio/cells-client/v2/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/v2/common.BuildRevision=${GITREV}"\
	 -o cec\
	 .

xgo:
	${GOPATH}/bin/xgo -go 1.15 \
	 --image pydio/xgo:latest \
	 --targets ${XGO_TARGETS} \
	 -ldflags "-X github.com/pydio/cells-client/v2/common.Version=${CELLS_CLIENT_VERSION}\
	 -X github.com/pydio/cells-client/v2/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/v2/common.BuildRevision=${GITREV}"\
	 -out cec\
	 ${PWD}


dev:
	go build\
	 -tags dev\
	 -ldflags "-X github.com/pydio/cells-client/v2/common.Version=${DEV_VERSION}\
	 -X github.com/pydio/cells-client/v2/common.BuildStamp=2021-01-01T00:00:00\
	 -X github.com/pydio/cells-client/v2/common.BuildRevision=dev"\
	 -o cec\
	 .

clean:
	rm -f cec	 