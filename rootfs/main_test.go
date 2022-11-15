package rootfs_test

import (
	"testing"
	rootfs "wsl/rootfs"

	"github.com/stretchr/testify/require"
)

func mockDownloadFile(mock func(string, string) (string, error)) (restore func()) {
	old := rootfs.DownloadFile
	rootfs.DownloadFile = mock
	return func() {
		rootfs.DownloadFile = old
	}
}

func TestGet(t *testing.T) {
	release := rootfs.Jammy

	dir := t.TempDir()
	tarball, err := rootfs.Get(release, dir)
	require.NoError(t, err)
	require.FileExists(t, tarball)

	// Testing proper caching
	oldDownload := rootfs.DownloadFile
	defer mockDownloadFile(func(destination string, url string) (string, error) {
		require.NotContains(t, url, rootfs.RemoteTarballName(release))
		return oldDownload(destination, url)
	})()
}
