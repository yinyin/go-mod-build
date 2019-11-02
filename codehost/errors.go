package codehost

import (
	"errors"
)

// ErrDirtyWorkCopy indicate work copy is dirty.
var ErrDirtyWorkCopy = errors.New("dirty work copy")

// ErrRecognizeRepo indicate failed to recognize repository type of given folder.
var ErrRecognizeRepo = errors.New("cannot recognize repository type")

// ErrNotRepo indicate given path is not repository of certain VCS.
type ErrNotRepo struct {
	VCSType string
	Path    string
}

func (e *ErrNotRepo) Error() string {
	return "not " + e.VCSType + " repo: [" + e.Path + "]"
}

func isErrNotRepo(err error) (isNotRepo bool) {
	_, isNotRepo = err.(*ErrNotRepo)
	return
}
