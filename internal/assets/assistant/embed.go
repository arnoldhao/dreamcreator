package assistantassets

import "embed"

//go:embed HatsuneMiku.vrm default.vrma
var FS embed.FS

const (
	BuiltinAvatarFile   = "HatsuneMiku.vrm"
	BuiltinMotionFile   = "default.vrma"
	BuiltinAvatarAssetID = "builtin:avatar:hatsune-miku"
	BuiltinMotionAssetID = "builtin:motion:default"
	BuiltinAvatarDisplayName = "Hatsune Miku"
	BuiltinMotionDisplayName = "Default Motion"
)
