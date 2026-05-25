package buildinfo

import "testing"

func TestCurrentReturnsGoVersion(t *testing.T) {
	info := Current()
	if info.GoVersion == "" {
		t.Fatal("expected go version to be set")
	}
}

func TestCurrentFallbackDoesNotPanic(t *testing.T) {
	prevVersion := versionOverride
	prevRevision := revisionOverride
	prevBuildTime := buildTimeOverride
	t.Cleanup(func() {
		versionOverride = prevVersion
		revisionOverride = prevRevision
		buildTimeOverride = prevBuildTime
	})

	versionOverride = ""
	revisionOverride = ""
	buildTimeOverride = ""
	info := Current()
	if info.Version == "" {
		t.Fatal("expected fallback version")
	}
	if info.Revision == "" {
		t.Fatal("expected fallback revision")
	}
	if info.RevisionShort == "" {
		t.Fatal("expected fallback revision short")
	}
}

func TestCurrentRevisionShortFromOverride(t *testing.T) {
	prevRevision := revisionOverride
	t.Cleanup(func() { revisionOverride = prevRevision })

	revisionOverride = "1234567890abcdef1234"
	info := Current()
	if info.Revision != revisionOverride {
		t.Fatalf("revision=%q", info.Revision)
	}
	if info.RevisionShort != "1234567890ab" {
		t.Fatalf("revision short=%q", info.RevisionShort)
	}
}
