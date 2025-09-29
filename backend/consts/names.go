package consts

import "runtime"

// AppDisplayName returns the human-friendly application name for the current OS.
func AppDisplayName() string {
	if runtime.GOOS == "windows" {
		return APP_NAME_WINDOWS
	}
	return APP_NAME
}

// AppDataDirName resolves the directory name used for storing user data on each OS.
func AppDataDirName() string {
	if runtime.GOOS == "windows" {
		return APP_NAME_WINDOWS
	}
	return APP_ID
}
