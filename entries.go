package gitgo

type Entries struct {
	Path string
	OID  string
	Stat string
}

func NewEntry(name, oid, stat string) *Entries {
	return &Entries{Path: name, OID: oid, Stat: stat}
}
