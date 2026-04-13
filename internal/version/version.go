// Package version 管理版本信息。
package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "0.1.0"
	Commit    = ""
	Date      = ""
	BuildInfo = ""
)

type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	OS        string
	Arch      string
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func String() string {
	info := Get()
	return fmt.Sprintf("Forge version %s\nCommit: %s\nBuilt: %s\nGo version: %s\nOS/Arch: %s/%s",
		info.Version, info.Commit, info.Date, info.GoVersion, info.OS, info.Arch)
}

func Print() {
	fmt.Println(String())
}

func IsDevelopment() bool {
	return Version == "" || Version == "dev"
}

func IsRelease() bool {
	return !IsDevelopment()
}
