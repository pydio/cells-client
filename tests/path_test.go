package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/pydio/cells-client/cmd"
)

type testingPaths struct {
	downloadTo       string
	downloadFrom     string
	downloadNodeFrom string
	expectedResult   string
}

//Download Cases
func TestSimpleTargetLocationPath(t *testing.T) {

	mock := &testingPaths{
		downloadTo:       "/Users/j/lulu",
		downloadFrom:     "personal-files/formula-one",
		downloadNodeFrom: "personal-files/formula-one/cars/redbull/verstappen.jpg",
		expectedResult:   "/Users/j/lulu/formula-one/cars/redbull/verstappen.jpg",
	}

	result := TargetLocation(mock.downloadTo, mock.downloadFrom, mock.downloadNodeFrom)
	assert.Equal(t, mock.expectedResult, result)
}

func TestComplicatedTargetLocationPath(t *testing.T) {

	mock := &testingPaths{
		downloadTo:       "/Users/j/lulu",
		downloadFrom:     "personal-files/formula-one/2019 season",
		downloadNodeFrom: "personal-files/formula-one/2019 season/tracks/monza/ferrari/vettel/first-corner.jpg",
		expectedResult:   "/Users/j/lulu/2019 season/tracks/monza/ferrari/vettel/first-corner.jpg",
	}
	result := TargetLocation(mock.downloadTo, mock.downloadFrom, mock.downloadNodeFrom)
	assert.Equal(t, mock.expectedResult, result)
}

func TestComplicatedSourceLocationPath(t *testing.T) {

	mock := &testingPaths{
		downloadTo:       "/Users/j/Downloads/my server/cells-test.your-files-your-rules.eu/",
		downloadFrom:     "personal-files/formula-one/2019 season",
		downloadNodeFrom: "personal-files/formula-one/2019 season/tracks/monza/ferrari/leclerc/first-corner.jpg",
		expectedResult:   "/Users/j/Downloads/my server/cells-test.your-files-your-rules.eu/2019 season/tracks/monza/ferrari/leclerc/first-corner.jpg",
	}
	result := TargetLocation(mock.downloadTo, mock.downloadFrom, mock.downloadNodeFrom)
	assert.Equal(t, mock.expectedResult, result)
}
