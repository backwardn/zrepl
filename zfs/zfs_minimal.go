package zfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type VersionType string

const (
	Bookmark VersionType = "bookmark"
	Snapshot VersionType = "snapshot"
)

func (t VersionType) DelimiterChar() string {
	switch t {
	case Bookmark:
		return "#"
	case Snapshot:
		return "@"
	default:
		panic(fmt.Sprintf("unexpected VersionType %#v", t))
	}
}

func (t VersionType) String() string {
	return string(t)
}

func DecomposeVersionString(v string) (fs string, versionType VersionType, name string, err error) {
	if len(v) < 3 {
		err = fmt.Errorf("snapshot or bookmark name implausibly short: %s", v)
		return
	}

	snapSplit := strings.SplitN(v, "@", 2)
	bookmarkSplit := strings.SplitN(v, "#", 2)
	if len(snapSplit)*len(bookmarkSplit) != 2 {
		err = fmt.Errorf("dataset cannot be snapshot and bookmark at the same time: %s", v)
		return
	}

	if len(snapSplit) == 2 {
		return snapSplit[0], Snapshot, snapSplit[1], nil
	} else {
		return bookmarkSplit[0], Bookmark, bookmarkSplit[1], nil
	}
}

type FilesystemVersion struct {
	Type VersionType

	// Display name. Should not be used for identification, only for user output
	Name string

	// GUID as exported by ZFS. Uniquely identifies a snapshot across pools
	Guid uint64

	// The TXG in which the snapshot was created. For bookmarks,
	// this is the GUID of the snapshot it was initially tied to.
	CreateTXG uint64

	// The time the dataset was created
	Creation time.Time
}

func (v FilesystemVersion) String() string {
	return fmt.Sprintf("%s%s", v.Type.DelimiterChar(), v.Name)
}

func (v FilesystemVersion) ToAbsPath(p *DatasetPath) string {
	var b bytes.Buffer
	b.WriteString(p.ToString())
	b.WriteString(v.Type.DelimiterChar())
	b.WriteString(v.Name)
	return b.String()
}

type StreamCopierError interface {
	error
	IsReadError() bool
	IsWriteError() bool
}

type StreamCopier interface {
	// WriteStreamTo writes the stream represented by this StreamCopier
	// to the given io.Writer.
	WriteStreamTo(w io.Writer) StreamCopierError
	// Close must be called as soon as it is clear that no more data will
	// be read from the StreamCopier.
	// If StreamCopier gets its data from a connection, it might hold
	// a lock on the connection until Close is called. Only closing ensures
	// that the connection can be used afterwards.
	Close() error
}

type DatasetFilter interface {
	Filter(p *DatasetPath) (pass bool, err error)
}

func ZFSListMapping(_ context.Context, _ DatasetFilter) ([]*DatasetPath, error) {
	panic("implement me")
}

func ZFSListFilesystemVersions(_ *DatasetPath, _ FilesystemVersionFilter) ([]FilesystemVersion, error) {
	panic("implement me")
}


type DrySendType string

const (
	DrySendTypeFull        DrySendType = "full"
	DrySendTypeIncremental DrySendType = "incremental"
)

func DrySendTypeFromString(s string) (DrySendType, error) {
	switch s {
	case string(DrySendTypeFull):
		return DrySendTypeFull, nil
	case string(DrySendTypeIncremental):
		return DrySendTypeIncremental, nil
	default:
		return "", fmt.Errorf("unknown dry send type %q", s)
	}
}

type DrySendInfo struct {
	Type         DrySendType
	Filesystem   string // parsed from To field
	From, To     string // direct copy from ZFS output
	SizeEstimate int64  // -1 if size estimate is not possible
}

func ZFSSendDry(_, _, _ string, _ string) (dsi DrySendInfo, err error) {
	panic("not impemented")
}

func ZFSSend(_ context.Context, _, _, _ string, _ string) (scp StreamCopier, err error) {
	panic("not impemented")
}

func ZFSGetReplicationCursor(_ *DatasetPath) (_ *FilesystemVersion, err error) {
	panic("not implemented")
}

func ZFSSetReplicationCursor(_ *DatasetPath, snap string) (_ uint64, err error) {
	panic("not implemented")
}


type FilesystemPlaceholderState struct {
	FS                    string
	FSExists              bool
	IsPlaceholder         bool
	RawLocalPropertyValue string
}

func ZFSGetFilesystemPlaceholderState(_ *DatasetPath) (phs FilesystemPlaceholderState, err error) {
	panic("not implemented")
}

func ZFSCreatePlaceholderFilesystem(_ *DatasetPath) error {
	panic("not implemented")
}

type RecvOptions struct {
	// Rollback to the oldest snapshot, destroy it, then perform `recv -F`.
	// Note that this doesn't change property values, i.e. an existing local property value will be kept.
	RollbackAndForceRecv bool
}

func ZFSSetPlaceholder(_ *DatasetPath, _ bool) error {
	panic("not implemenmted")
}

func ZFSRecv(_ context.Context, _ string, _ StreamCopier, _ RecvOptions) error {
	panic("not implemented")
}

func ZFSDestroyFilesystemVersion(_ *DatasetPath, version *FilesystemVersion) error {
	panic("not implemented")
}

type FilesystemVersionFilter interface {
	Filter(t VersionType, name string) (accept bool, err error)
}

func ZFSSnapshot(_ *DatasetPath, _ string, _ bool) error {
	panic("not implemented")
}

func PrometheusRegister(registry prometheus.Registerer) error {
	panic("not implemented")
}

func ZFSList(_ []string) ([][]string, error) {
	panic("not implemented")
}

const (
	// For a placeholder filesystem to be a placeholder, the property source must be local,
	// i.e. not inherited.
	PlaceholderPropertyName string = "zrepl:placeholder"
	placeholderPropertyOn   string = "on"
	placeholderPropertyOff  string = "off"
)

func NoFilter() DatasetFilter {
	panic("not implemetned")
}


type MigrateHashBasedPlaceholderReport struct {
	OriginalState     FilesystemPlaceholderState
	NeedsModification bool
}

// fs must exist, will panic otherwise
func ZFSMigrateHashBasedPlaceholderToCurrent(fs *DatasetPath, dryRun bool) (*MigrateHashBasedPlaceholderReport, error) {
	panic("not implemented")
}
