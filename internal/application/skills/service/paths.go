package service

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	dreamCreatorAppDirName        = "dreamcreator"
	dreamCreatorWorkspacesDirName = "workspaces"
	dreamCreatorSkillsDirName     = "skills"
)

func resolveDreamCreatorAppRoot(workspaceRoot string) string {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return ""
		}
		return filepath.Join(configDir, dreamCreatorAppDirName)
	}
	if absolute, err := filepath.Abs(root); err == nil {
		root = absolute
	}
	parent := filepath.Dir(root)
	if strings.EqualFold(filepath.Base(parent), dreamCreatorWorkspacesDirName) {
		return filepath.Dir(parent)
	}
	return parent
}

func resolveDreamCreatorSkillsRoot(workspaceRoot string) string {
	appRoot := resolveDreamCreatorAppRoot(workspaceRoot)
	if appRoot == "" {
		return ""
	}
	return filepath.Join(appRoot, dreamCreatorSkillsDirName)
}

func resolveSkillDocumentPath(skillID string, workspaceRoot string) string {
	trimmed := strings.TrimSpace(skillID)
	if trimmed == "" {
		return ""
	}
	root := resolveDreamCreatorSkillsRoot(workspaceRoot)
	if root == "" {
		return filepath.ToSlash(filepath.Join(dreamCreatorSkillsDirName, trimmed, "SKILL.md"))
	}
	return filepath.ToSlash(filepath.Join(root, trimmed, "SKILL.md"))
}

func ResolveSkillDocumentPath(skillID string, workspaceRoot string) string {
	return resolveSkillDocumentPath(skillID, workspaceRoot)
}
