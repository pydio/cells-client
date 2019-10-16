ENV=env GOOS=linux
TODAY=`date -u +%Y-%m-%dT%H:%M:%S`
GITREV=`git rev-parse HEAD`

main:
	go build -a\
	 -ldflags "-X github.com/pydio/cells-client/common.Version=${CELLS_VERSION}\
	 -X github.com/pydio/cells-client/common.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/common.BuildRevision=${GITREV}\
	 -o cec\
	 .

dev:
	go build\
	 -tags dev\
	 -ldflags "-X github.com/pydio/cells-client/common.Version=0.2.0\
	 -X github.com/pydio/cells-client/common.BuildStamp=2018-01-01T00:00:00\
	 -X github.com/pydio/cells-client/common.BuildRevision=dev"\
	 -o cec\
	 .

clean:
	rm -f cec	 