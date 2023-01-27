package gowsl

// This contains shared windows API definitions for the real and mock implementations.

// Types of file in Windows
// https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfiletype
type fileType int

const (
	fileTypeChar    fileType = 0x0002 // The specified file is a character file, typically an LPT device or a console.
	fileTypeDisk    fileType = 0x0001 // The specified file is a disk file.
	fileTypePipe    fileType = 0x0003 // The specified file is a socket, a named pipe, or an anonymous pipe.
	fileTypeRemote  fileType = 0x8000 // Unused.
	fileTypeUnknown fileType = 0x0000 // Either the type of the specified file is unknown, or the function failed.
)
