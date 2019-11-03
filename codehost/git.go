package codehost

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const gitCommandName = "git"

func prepareGitEnv() (result []string) {
	osEnv := os.Environ()
	result = make([]string, 0, len(osEnv))
	for _, envVal := range osEnv {
		if !strings.HasPrefix(envVal, "TZ=") {
			result = append(result, envVal)
		}
	}
	result = append(result, "TZ=UTC")
	return
}

// GitRepo represent code repo in git.
type GitRepo struct {
	ModPath string
	gitPath string
	gitEnv  []string
}

// NewGitRepo create new instance of GitRepo with given module path.
func NewGitRepo(modPath string) (repo *GitRepo, err error) {
	repo = &GitRepo{
		ModPath: modPath,
	}
	if err = repo.checkClean(); nil != err {
		return nil, err
	}
	return
}

func (repo *GitRepo) gitCmd(arg ...string) (cmd *exec.Cmd, err error) {
	if repo.gitPath == "" {
		if repo.gitPath, err = exec.LookPath(gitCommandName); err != nil {
			return
		}
	}
	if len(repo.gitEnv) == 0 {
		repo.gitEnv = prepareGitEnv()
	}
	cmd = &exec.Cmd{
		Path: repo.gitPath,
		Args: append([]string{gitCommandName}, arg...),
		Env:  repo.gitEnv,
		Dir:  repo.ModPath,
	}
	return
}

func (repo *GitRepo) checkClean() (err error) {
	var cmd *exec.Cmd
	if cmd, err = repo.gitCmd("diff", "--no-ext-diff", "--quiet", "--exit-code"); nil != err {
		return
	}
	if err = cmd.Run(); nil != err {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			return
		}
		switch ee.ExitCode() {
		case 1:
			return ErrDirtyWorkCopy
		case 129:
			fallthrough
		case 128:
			return &ErrNotRepo{
				VCSType: "git",
				Path:    repo.ModPath,
			}
		}
		return
	}
	return nil
}

func (repo *GitRepo) workingTag() (tag string, err error) {
	var cmd *exec.Cmd
	if cmd, err = repo.gitCmd("ls-remote", "--quiet", "./."); nil != err {
		return
	}
	buf, err := cmd.Output()
	if nil != err {
		return
	}
	var headHash string
	var tagRefs []*tagRef
	for _, line := range strings.Split(string(buf), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		refHash := f[0]
		refName := f[1]
		if refName == "HEAD" {
			headHash = refHash
		} else if strings.HasPrefix(refName, "refs/tags/v") {
			tagName := refName[len("refs/tags/"):]
			tagName = strings.TrimSuffix(tagName, "^{}")
			if !semver.IsValid(tagName) {
				continue
			}
			tagRefs = append(tagRefs, &tagRef{
				Tag:  tagName,
				Hash: refHash,
			})
		}
	}
	if headHash == "" {
		return
	}
	for _, ref := range tagRefs {
		if ref.Hash != headHash {
			continue
		}
		if (tag == "") || (semver.Compare(ref.Tag, tag) > 0) {
			tag = ref.Tag
		}
	}
	return
}

// PseudoVersion generate pseudo version.
func (repo *GitRepo) PseudoVersion() (pseudoVersion string, err error) {
	tag, err := repo.workingTag()
	if nil != err {
		return
	}
	if tag != "" {
		return tag, nil
	}
	var cmd *exec.Cmd
	if cmd, err = repo.gitCmd("show", "--no-patch", "--pretty=format:v0.0.0-%cd-%h", "--date=format-local:%Y%m%d%H%M%S", "--abbrev=12"); nil != err {
		return
	}
	buf, err := cmd.Output()
	if nil != err {
		return
	}
	return string(bytes.TrimSpace(buf)), nil
}

// CommitTime get the commit time of current workcopy.
func (repo *GitRepo) CommitTime() (commitTime time.Time, err error) {
	var cmd *exec.Cmd
	if cmd, err = repo.gitCmd("show", "--no-patch", "--pretty=format:%ct"); nil != err {
		return
	}
	buf, err := cmd.Output()
	if nil != err {
		return
	}
	epoch, err := strconv.ParseInt(string(bytes.TrimSpace(buf)), 10, 64)
	if nil != err {
		return
	}
	commitTime = time.Unix(epoch, 0).UTC()
	return
}

// Zip create module zip file.
func (repo *GitRepo) Zip(w io.Writer, modulePath, moduleVersion string) (err error) {
	tmpdir, err := ioutil.TempDir("", "go-mod-pack-codehost-git")
	if nil != err {
		log.Printf("ERROR: setup temporary folder for git archive failed: %v", err)
		return
	}
	defer os.RemoveAll(tmpdir)
	tmpzipfile := filepath.Join(tmpdir, "git-archive.zip")
	var cmd *exec.Cmd
	if cmd, err = repo.gitCmd("archive", "--format=zip", "--output", tmpzipfile, "HEAD"); nil != err {
		return
	}
	if err = cmd.Run(); nil != err {
		return
	}
	err = convertToModuleZip(w, modulePath, moduleVersion, tmpzipfile)
	return
}
