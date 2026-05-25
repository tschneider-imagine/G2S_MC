package buildinfo

import (
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	versionOverride   string
	revisionOverride  string
	buildTimeOverride string
)

type RuntimeInfo struct {
	Version       string `json:"version"`
	Revision      string `json:"revision"`
	RevisionShort string `json:"revision_short"`
	Modified      bool   `json:"modified"`
	BuildTime     string `json:"build_time"`
	GoVersion     string `json:"go_version"`
}

func Current() RuntimeInfo {
	info := RuntimeInfo{
		Version:   "dev",
		Revision:  "unknown",
		BuildTime: "unknown",
		GoVersion: runtime.Version(),
	}
	if strings.TrimSpace(versionOverride) != "" {
		info.Version = strings.TrimSpace(versionOverride)
	}
	if strings.TrimSpace(revisionOverride) != "" {
		info.Revision = strings.TrimSpace(revisionOverride)
	}
	if strings.TrimSpace(buildTimeOverride) != "" {
		info.BuildTime = strings.TrimSpace(buildTimeOverride)
	}

	if build, ok := debug.ReadBuildInfo(); ok {
		if strings.TrimSpace(build.GoVersion) != "" {
			info.GoVersion = strings.TrimSpace(build.GoVersion)
		}
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if strings.TrimSpace(revisionOverride) == "" && strings.TrimSpace(setting.Value) != "" {
					info.Revision = strings.TrimSpace(setting.Value)
				}
			case "vcs.modified":
				info.Modified = strings.EqualFold(strings.TrimSpace(setting.Value), "true")
			case "vcs.time":
				if strings.TrimSpace(buildTimeOverride) == "" && strings.TrimSpace(setting.Value) != "" {
					info.BuildTime = strings.TrimSpace(setting.Value)
				}
			}
		}
	}

	if strings.TrimSpace(info.Revision) != "" && !strings.EqualFold(info.Revision, "unknown") {
		if len(info.Revision) > 12 {
			info.RevisionShort = info.Revision[:12]
		} else {
			info.RevisionShort = info.Revision
		}
	} else {
		info.RevisionShort = "unknown"
	}

	return info
}
