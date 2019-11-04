ENV=env GOOS=linux
TODAY:=$(shell date -u +%Y-%m-%dT%H:%M:%S)
GITREV:=$(shell git rev-parse HEAD)
CELLS_CLIENT_VERSION?=0.2.0
XGO_TARGETS?="linux/amd64,darwin/amd64,windows/amd64"


main:
	go build -a\
	 -ldflags "-X github.com/pydio/cells-client/common.Version=${CELLS_CLIENT_VERSION}\
	 -X github.com/pydio/cells-client/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/common.BuildRevision=${GITREV}"\
	 -o cec\
	 .

xgo:
	${GOPATH}/bin/xgo -go 1.12 \
	 --image pydio/xgo:latest \
	 --targets ${XGO_TARGETS} \
	 -ldflags "-X github.com/pydio/cells-client/common.Version=${CELLS_CLIENT_VERSION}\
	 -X github.com/pydio/cells-client/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/common.BuildRevision=${GITREV}"\
	 -out cec\
	 ${PWD}


dev:
	go build\
	 -tags dev\
	 -ldflags "-X github.com/pydio/cells-client/common.Version=${CELLS_CLIENT_VERSION}\
	 -X github.com/pydio/cells-client/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/common.BuildRevision=${GITREV}"\
	 -o cec\
	 .