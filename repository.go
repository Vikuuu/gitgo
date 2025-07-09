package gitgo

import "path/filepath"

type Repository struct {
	Path     string
	GitPath  string
	Database string
	Index    string
	Refs     string
}

func NewRepository(path string) Repository {
	return Repository{
		Path:     path,
		GitPath:  filepath.Join(path, ".gitgo"),
		Database: filepath.Join(path, ".gitgo", "objects"),
		Index:    filepath.Join(path, ".gitgo", "index"),
		Refs:     filepath.Join(path, ".gitgo"),
	}
}
