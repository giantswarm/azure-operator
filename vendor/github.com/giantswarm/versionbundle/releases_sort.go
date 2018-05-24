package versionbundle

import (
	"time"

	"github.com/coreos/go-semver/semver"
)

type SortReleasesByVersion []Release

func (r SortReleasesByVersion) Len() int      { return len(r) }
func (r SortReleasesByVersion) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r SortReleasesByVersion) Less(i, j int) bool {
	verA := semver.New(r[i].Version())
	verB := semver.New(r[j].Version())
	return verA.LessThan(*verB)
}

type SortReleasesByTimestamp []Release

func (r SortReleasesByTimestamp) Len() int      { return len(r) }
func (r SortReleasesByTimestamp) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r SortReleasesByTimestamp) Less(i, j int) bool {
	iTime, err := time.Parse(releaseTimestampFormat, r[i].Timestamp())
	if err != nil {
		panic(err)
	}

	jTime, err := time.Parse(releaseTimestampFormat, r[j].Timestamp())
	if err != nil {
		panic(err)
	}

	return iTime.UnixNano() < jTime.UnixNano()
}
