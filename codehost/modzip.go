package codehost

import (
	"archive/zip"
	"io"
	"os"
	"strings"

	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"
)

// convertToModuleZip translate given VCS zip into module zip.
func convertToModuleZip(w io.Writer, zipfilepath string, modulePath, moduleVersion string) (err error) {
	zr, err := zip.OpenReader(zipfilepath)
	if nil != err {
		return
	}
	defer zr.Close()
	var files []modzip.File
	for _, zf := range zr.File {
		if (zf.Name == "") || strings.HasSuffix(zf.Name, "/") {
			continue
		}
		files = append(files, &modZipFile{
			name: zf.Name,
			f:    zf,
		})
	}
	err = modzip.Create(w, module.Version{
		Path:    modulePath,
		Version: moduleVersion,
	}, files)
	return
}

type modZipFile struct {
	name string
	f    *zip.File
}

func (f modZipFile) Path() string {
	return f.name
}

func (f modZipFile) Lstat() (os.FileInfo, error) {
	return f.f.FileInfo(), nil
}

func (f modZipFile) Open() (io.ReadCloser, error) {
	return f.f.Open()
}
