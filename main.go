package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	rungo "github.com/yinyin/go-run-go"

	"github.com/yinyin/go-mod-pack/codehost"
	"github.com/yinyin/go-mod-pack/modproxyfolder"
)

func fetchTargetModuleInfo(cmdGo *rungo.CommandGo) (modInfo *rungo.Module, err error) {
	modInfos, err := cmdGo.ListModule()
	if nil != err {
		err = fmt.Errorf("invoke list module failed: %w", err)
		return
	}
	if l := len(modInfos); l < 1 {
		err = errors.New("empty module list")
		return
	} else if l != 1 {
		log.Printf("WARN: got multiple module information. only 1st one will be pack.")
		for idx, mInfo := range modInfos {
			if idx == 0 {
				log.Printf("* %s@%s (will pack this one)", mInfo.Path, mInfo.Version)
			} else {
				log.Printf("- %s@%s", mInfo.Path, mInfo.Version)
			}
		}
	}
	modInfo = modInfos[0]
	if modInfo.Path == "" {
		err = errors.New("empty module path")
	}
	if modInfo.Dir == "" {
		err = errors.New("empty module folder path")
	}
	if modInfo.GoMod == "" {
		err = errors.New("empty module definition path")
	}
	return
}

func parseOptions() (force bool) {
	flag.BoolVar(&force, "force", false, "force module packaging")
	flag.Parse()
	return
}

func main() {
	force := parseOptions()
	cmdGo := rungo.CommandGo{}
	modInfo, err := fetchTargetModuleInfo(&cmdGo)
	if nil != err {
		log.Fatalf("ERROR: cannot have module info: %v", err)
		return
	}
	log.Printf("INFO: module path [%s]", modInfo.Path)
	repo, err := codehost.NewRepo(modInfo.Dir)
	if nil != err {
		log.Fatalf("ERROR: cannot open repository: %v", err)
		return
	}
	pseudoVer, err := repo.PseudoVersion()
	if nil != err {
		log.Fatalf("ERROR: fetch pseudo version failed: %v", err)
		return
	}
	modProxyFolder, err := modproxyfolder.NewModuleProxyFolder("", modInfo.Path)
	if nil != err {
		log.Fatalf("ERROR: cannot setup module proxy folder data: %v", err)
		return
	}
	log.Printf("INFO: module proxy folder: %s", modProxyFolder.FolderPath)
	hasVersion, err := modProxyFolder.ContainVersion(pseudoVer)
	if nil != err {
		log.Fatalf("ERROR: check version (%s) exist failed: %v", pseudoVer, err)
		return
	}
	if hasVersion {
		if !force {
			log.Printf("INFO: version existed in version list: %s. Stopped.", pseudoVer)
			return
		}
		log.Printf("WARN: version existed (%s). proceed packaging as force mode is enabled.", pseudoVer)
	}
	commitTime, err := repo.CommitTime()
	if nil != err {
		log.Printf("ERROR: fetch commit time failed: %v", err)
		return
	}
	log.Printf("INFO: commit at %v", commitTime)
	if err = modProxyFolder.SaveInfo(modproxyfolder.Info{
		Version: pseudoVer,
		Time:    commitTime,
	}); nil != err {
		log.Printf("ERROR: write module info failed: %v", err)
		return
	}
	srcGoModFile, err := os.Open(modInfo.GoMod)
	log.Printf("INFO: source go.mod: %v", modInfo.GoMod)
	if nil != err {
		log.Fatalf("ERROR: open source go.mod (%s) for read failed: %v", modInfo.GoMod, err)
		return
	}
	defer srcGoModFile.Close()
	modFile, err := modProxyFolder.CreateGoMod(pseudoVer)
	if nil != err {
		log.Fatalf("ERROR: open .mod file failed: %v", err)
		return
	}
	defer modFile.Close()
	if _, err = io.Copy(modFile, srcGoModFile); nil != err {
		log.Fatalf("ERROR: copy go.mod failed: %v", err)
		return
	}
	zipFile, err := modProxyFolder.CreateZip(pseudoVer)
	if nil != err {
		log.Fatalf("ERROR: open .zip file failed: %v", err)
		return
	}
	defer zipFile.Close()
	if err = repo.Zip(zipFile, modInfo.Path, pseudoVer); nil != err {
		log.Fatalf("ERROR: create module zip failed: %v", err)
		return
	}
	if err = modProxyFolder.AddVersionToList(pseudoVer); nil != err {
		log.Fatalf("ERROR: add version into list failed: %v", err)
		return
	}
	log.Print("Complete.")
	return
}
