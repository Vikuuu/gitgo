package gitgo

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
)

type Node interface{}

type Tree struct {
	Nodes map[string]Node
}

func NewTree() *Tree {
	return &Tree{Nodes: make(map[string]Node)}
}

func BuildTree(entries []Entries) *Tree {
	root := NewTree()
	for _, entry := range entries {
		parts := strings.Split(entry.Path, "/")
		current := root
		// Walk through each part of the path.
		for i, part := range parts {
			// Last part: this is a file entry.
			if i == len(parts)-1 {
				current.Nodes[part] = NewEntry(part, entry.OID, entry.Stat)
			} else {
				// Intermediate directory: see if a Tree already exists.
				if node, ok := current.Nodes[part]; ok {
					if subtree, ok := node.(*Tree); ok {
						current = subtree
					} else {
						// Conflict: an entry already exists where a directory is expected.
						log.Printf("%q already exists as a file; cannot create directory\n", part)
						break
					}
				} else {
					// Create a new tree for this directory and continue.
					newTree := NewTree()
					current.Nodes[part] = newTree
					current = newTree
				}
			}
		}
	}
	return root
}

func TraverseTree(tree *Tree) ([]Entries, error) {
	entry := []Entries{}
	for name, node := range tree.Nodes {
		switch n := node.(type) {
		case *Tree:
			e, err := TraverseTree(n)
			if err != nil {
				return nil, err
			}
			tree := TreeBlob{Data: CreateTreeEntry(e)}.Init()
			hash, err := tree.Store()
			if err != nil {
				return nil, err
			}
			ne := Entries{Path: name, OID: hash, Stat: "040000"}
			entry = append(entry, ne)
		case *Entries:
			entry = append(entry, *n)
		default:
			return nil, errors.New("unknown data type")
		}
	}
	return entry, nil
}

func CreateTreeEntry(entries []Entries) bytes.Buffer {
	var buf bytes.Buffer
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	for _, entry := range entries {
		input := fmt.Sprintf("%s %s", entry.Stat, entry.Path)
		buf.WriteString(input)
		buf.WriteByte(0)
		rawOID, err := hex.DecodeString(entry.OID)
		if err != nil {
			log.Printf("Error: coverting oid to raw: %s, for : %s", err, entry.Path)
		}
		buf.Write(rawOID)
	}
	return buf
}
