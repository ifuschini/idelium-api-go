package buildinfo

import "testing"

func TestCurrentReturnsInjectedBuildIdentity(t *testing.T) {
	previousVersion, previousCommit := Version, Commit
	t.Cleanup(func() {
		Version, Commit = previousVersion, previousCommit
	})
	Version = "1.2.3-test"
	Commit = "0123456789abcdef"

	info := Current()
	if info.Service != "idelium-api-go" || info.Version != Version || info.Commit != Commit {
		t.Fatalf("unexpected build identity: %#v", info)
	}
}
