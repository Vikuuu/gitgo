package gitgo

import (
	"os"
	"syscall"
)

const (
	maxPathSize    = 0xfff
	regularMode    = 0100644
	executableMode = 0100755
	headerFormat   = "a4N2"
)

type Entries struct {
	Path string
	OID  string
	Stat string
}

func NewEntry(name, oid, stat string) *Entries {
	return &Entries{Path: name, OID: oid, Stat: stat}
}

type IndexEntry struct {
	Path      string
	Oid       string
	Mtime     int64
	MtimeNsec int64
	Ctime     int64
	CtimeNsec int64
	Dev       uint64
	Ino       uint64
	Mode      int
	Uid       uint32
	Gid       uint32
	Size      int64
	Flags     uint32
}

func modeForStat(s os.FileInfo) int {
	// stat.Mode().IsRegular()
	if s.Mode()&0111 != 0 {
		return executableMode
	} else {
		return regularMode
	}
}

func NewIndexEntry(name, oid string, stat os.FileInfo) *IndexEntry {
	s := stat.Sys().(*syscall.Stat_t)
	flags := min(len(name), maxPathSize)
	m := modeForStat(stat)
	return &IndexEntry{
		Path:      name,
		Oid:       oid,
		Mtime:     s.Mtim.Sec,
		MtimeNsec: s.Mtim.Nsec,
		Ctime:     s.Ctim.Sec,
		CtimeNsec: s.Ctim.Nsec,
		Dev:       s.Dev,
		Ino:       s.Ino,
		Mode:      m,
		Uid:       s.Uid,
		Gid:       s.Gid,
		Size:      s.Size,
		Flags:     uint32(flags),
	}
}

func (ie IndexEntry) StatMatch(stat os.FileInfo) bool {
	return ie.Mode == modeForStat(stat) && (ie.Size == 0 || ie.Size == stat.Size())
}
