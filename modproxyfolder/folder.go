package modproxyfolder

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"

	rungo "github.com/yinyin/go-run-go"
)

const (
	moduleVersFolderName = "@v"
	moduleListFileName   = "list"
	moduleInfoFileSuffix = "info"
	moduleModFileSuffix  = "mod"
	moduleZipFileSuffix  = "zip"
)

// ErrEmptyVersion indicate given version is empty.
var ErrEmptyVersion = errors.New("given version is empty")

// DefaultModuleProxyBaseFolder fetch `GOPATH` from `go env` and generate
// default base folder path of module store (${GOPATH}/pkg/mod/cache/download).
func DefaultModuleProxyBaseFolder() (baseFolder string, err error) {
	cmdGo := rungo.CommandGo{}
	envInfo, err := cmdGo.Env()
	if nil != err {
		return
	}
	baseFolder = filepath.Join(envInfo.GoPath, "pkg/mod/cache/download")
	return
}

// ModuleProxyFolder represent folder of a module.
type ModuleProxyFolder struct {
	FolderPath string
	ModulePath string

	hasVersFolder bool
}

// NewModuleProxyFolder create an instance of ModuleProxyFolder.
// If baseFolder is empty, default path based on GOPATH (${GOPATH}/pkg/mod/cache/download)
// will be generated.
func NewModuleProxyFolder(baseFolder string, modulePath string) (modProxyFolder *ModuleProxyFolder, err error) {
	if baseFolder == "" {
		if baseFolder, err = DefaultModuleProxyBaseFolder(); nil != err {
			log.Printf("ERROR: unable to fetch GOPATH with `go env`: %v", err)
			return
		}
	}
	escapedModPath, err := module.EscapePath(modulePath)
	if nil != err {
		return
	}
	modProxyFolder = &ModuleProxyFolder{
		FolderPath: filepath.Join(baseFolder, escapedModPath),
		ModulePath: modulePath,
	}
	return
}

func (f *ModuleProxyFolder) prepareVersFolder() (err error) {
	if f.hasVersFolder {
		return
	}
	p := filepath.Join(f.FolderPath, moduleVersFolderName)
	if _, err = os.Stat(p); os.IsNotExist(err) {
		err = os.MkdirAll(p, 0755)
	}
	if nil != err {
		return
	}
	f.hasVersFolder = true
	return
}

// LoadVersionList fetch versions from list file.
func (f *ModuleProxyFolder) LoadVersionList() (vers []module.Version, err error) {
	listFilePath := filepath.Join(f.FolderPath, moduleVersFolderName, moduleListFileName)
	fp, err := os.Open(listFilePath)
	if nil != err {
		return
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		v := strings.TrimSpace(scanner.Text())
		if v == "" {
			continue
		}
		vers = append(vers, module.Version{
			Path:    f.ModulePath,
			Version: v,
		})
	}
	if err = scanner.Err(); err != nil {
		return
	}
	return
}

// SaveVersionList write given versions into list file.
func (f *ModuleProxyFolder) SaveVersionList(vers []module.Version) (err error) {
	module.Sort(vers)
	if err = f.prepareVersFolder(); nil != err {
		return
	}
	listFilePath := filepath.Join(f.FolderPath, moduleVersFolderName, moduleListFileName)
	fp, err := os.Create(listFilePath)
	if nil != err {
		return
	}
	defer fp.Close()
	for _, v := range vers {
		if _, err = fp.WriteString(v.Version + "\n"); nil != err {
			return
		}
	}
	return nil
}

// AddVersionToList add given version into list file.
func (f *ModuleProxyFolder) AddVersionToList(ver string) (err error) {
	vers, err := f.LoadVersionList()
	if nil != err {
		return
	}
	for _, v := range vers {
		if v.Version == ver {
			return
		}
	}
	vers = append(vers, module.Version{
		Path:    f.ModulePath,
		Version: ver,
	})
	return f.SaveVersionList(vers)
}

// ImportVersionsToList feed version strings into list file
func (f *ModuleProxyFolder) ImportVersionsToList(versions []string) (err error) {
	vers, err := f.LoadVersionList()
	if nil != err {
		return
	}
	m := make(map[string]struct{})
	for _, v := range vers {
		m[v.Version] = struct{}{}
	}
	for _, vText := range versions {
		if _, ok := m[vText]; !ok {
			vers = append(vers, module.Version{
				Path:    f.ModulePath,
				Version: vText,
			})
		}
	}
	return f.SaveVersionList(vers)
}

// ContainVersion check if given version is contain in folder.
func (f *ModuleProxyFolder) ContainVersion(ver string) (hasVersion bool, err error) {
	vers, err := f.LoadVersionList()
	if nil != err {
		return
	}
	for _, v := range vers {
		if v.Version == ver {
			return true, nil
		}
	}
	return false, nil
}

func (f *ModuleProxyFolder) createVersionedFile(ver, suffix string) (fp *os.File, err error) {
	escapedVer, err := module.EscapeVersion(ver)
	if nil != err {
		return
	}
	if err = f.prepareVersFolder(); nil != err {
		return
	}
	p := filepath.Join(f.FolderPath, moduleVersFolderName, escapedVer+"."+suffix)
	return os.Create(p)
}

// SaveInfo store the given info.
func (f *ModuleProxyFolder) SaveInfo(info Info) (err error) {
	if info.Version == "" {
		err = ErrEmptyVersion
		return
	}
	info.Time = info.Time.UTC()
	buf, err := json.Marshal(&info)
	if nil != err {
		return
	}
	fp, err := f.createVersionedFile(info.Version, moduleInfoFileSuffix)
	if nil != err {
		return
	}
	defer fp.Close()
	_, err = fp.Write(buf)
	return
}

// CreateGoMod open the mod file for write.
func (f *ModuleProxyFolder) CreateGoMod(ver string) (fp *os.File, err error) {
	if ver == "" {
		err = ErrEmptyVersion
		return
	}
	return f.createVersionedFile(ver, moduleModFileSuffix)
}

// CreateZip open module zip file for write.
func (f *ModuleProxyFolder) CreateZip(ver string) (fp *os.File, err error) {
	if ver == "" {
		err = ErrEmptyVersion
		return
	}
	return f.createVersionedFile(ver, moduleZipFileSuffix)
}
