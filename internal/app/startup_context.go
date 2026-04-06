package app

import "strings"

const autoStartLaunchArgument = "--autostart"

type startupContext struct {
	launchedByAutoStart bool
}

func currentStartupContext(args []string) startupContext {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), autoStartLaunchArgument) {
			return startupContext{launchedByAutoStart: true}
		}
	}
	return startupContext{}
}
