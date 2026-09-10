//go:build windows

package dedupe

import (
	"os"

	"golang.org/x/sys/windows"
)

type fileIdentity struct {
	volume uint64
	index  uint64
}

func getFileIdentity(path string, _ os.FileInfo) (fileIdentity, bool) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fileIdentity{}, false
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return fileIdentity{}, false
	}
	defer windows.CloseHandle(handle)

	var data windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &data); err != nil {
		return fileIdentity{}, false
	}
	index := uint64(data.FileIndexHigh)<<32 | uint64(data.FileIndexLow)
	return fileIdentity{volume: uint64(data.VolumeSerialNumber), index: index}, true
}
