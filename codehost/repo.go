package codehost

import (
	"io"
)

// Repo defines interface of operations in VCS repository.
type Repo interface {
	// PseudoVersion generate pseudo version.
	PseudoVersion() (pseudoVersion string, err error)

	// Zip create module zip file.
	Zip(w io.Writer, modulePath, moduleVersion string) (err error)
}

// NewRepo create a Repo base on given module folder path.
func NewRepo(modFolderPath string) (repo Repo, err error) {
	if repo, err = NewGitRepo(modFolderPath); (nil == err) || (nil != err && !isErrNotRepo(err)) {
		return
	}
	return nil, ErrRecognizeRepo
}
