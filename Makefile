ENV=env GOOS=linux
TODAY=`date -u +%Y-%m-%dT%H:%M:%S`
GITREV=`git rev-parse HEAD`

main:
	go build -a\
	 -ldflags "-X github.com/pydio/cells-client/cmd.version=${CELLS_VERSION}\
	 -X github.com/pydio/cells-client/cmd.BuildStamp=${TODAY}\
	 -X github.com/pydio/cells-client/cmd.BuildRevision=${GITREV}\
	 -o cec\
	 .

dev:
	go build\
	 -tags dev\
	 -ldflags "-X github.com/pydio/cells-client/cmd.version=0.2.0\
	 -X github.com/pydio/cells-client/cmd.BuildStamp=2018-01-01T00:00:00\
	 -X github.com/pydio/cells-client/cmd.BuildRevision=dev"\
	 -o cec\
	 .