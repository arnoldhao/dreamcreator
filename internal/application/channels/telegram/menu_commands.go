package telegram

import (
	"regexp"
	"strings"
	"unicode/utf8"

	appcommands "dreamcreator/internal/application/commands"
)

const TelegramMaxCommands = 100
const telegramCommandDescriptionMax = 256

var telegramCommandNamePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

var telegramNativeP2CommandKeys = map[string]struct{}{
	"restart":  {},
	"config":   {},
	"debug":    {},
	"exec":     {},
	"elevated": {},
}

type MenuCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type CustomCommandInput struct {
	Command     string
	Description string
}

func IsTelegramP2NativeCommandKey(key string) bool {
	_, ok := telegramNativeP2CommandKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

func FilterTelegramMenuNativeCommandSpecs(specs []appcommands.NativeCommandSpec) []appcommands.NativeCommandSpec {
	if len(specs) == 0 {
		return nil
	}
	filtered := make([]appcommands.NativeCommandSpec, 0, len(specs))
	for _, spec := range specs {
		if IsTelegramP2NativeCommandKey(spec.Key) {
			continue
		}
		filtered = append(filtered, spec)
	}
	return filtered
}

type CustomCommandIssue struct {
	Index   int
	Field   string
	Message string
}

func NormalizeTelegramCommandName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		trimmed = trimmed[1:]
	}
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(trimmed, "-", "_")))
}

func NormalizeTelegramCommandDescription(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) <= telegramCommandDescriptionMax {
		return trimmed
	}
	return truncateTelegramDescription(trimmed, telegramCommandDescriptionMax)
}

func truncateTelegramDescription(value string, limit int) string {
	if limit <= 0 || value == "" {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	count := 0
	index := 0
	for i, r := range value {
		if count == limit {
			break
		}
		index = i + utf8.RuneLen(r)
		count++
	}
	return strings.TrimSpace(value[:index])
}

func ResolveTelegramCustomCommands(params ResolveCustomCommandsParams) ResolveCustomCommandsResult {
	entries := params.Commands
	reserved := params.ReservedCommands
	if reserved == nil {
		reserved = make(map[string]struct{})
	}
	checkReserved := params.CheckReserved
	if !params.CheckReservedSet {
		checkReserved = true
	}
	checkDuplicates := params.CheckDuplicates
	if !params.CheckDuplicatesSet {
		checkDuplicates = true
	}
	seen := make(map[string]struct{})
	resolved := make([]MenuCommand, 0, len(entries))
	issues := make([]CustomCommandIssue, 0)

	for index, entry := range entries {
		normalized := NormalizeTelegramCommandName(entry.Command)
		if normalized == "" {
			issues = append(issues, CustomCommandIssue{
				Index:   index,
				Field:   "command",
				Message: "Telegram custom command is missing a command name.",
			})
			continue
		}
		if !telegramCommandNamePattern.MatchString(normalized) {
			issues = append(issues, CustomCommandIssue{
				Index:   index,
				Field:   "command",
				Message: "Telegram custom command is invalid (use a-z, 0-9, underscore; max 32 chars).",
			})
			continue
		}
		if checkReserved {
			if _, ok := reserved[normalized]; ok {
				issues = append(issues, CustomCommandIssue{
					Index:   index,
					Field:   "command",
					Message: "Telegram custom command conflicts with a native command.",
				})
				continue
			}
		}
		if checkDuplicates {
			if _, ok := seen[normalized]; ok {
				issues = append(issues, CustomCommandIssue{
					Index:   index,
					Field:   "command",
					Message: "Telegram custom command is duplicated.",
				})
				continue
			}
		}
		description := NormalizeTelegramCommandDescription(entry.Description)
		if description == "" {
			issues = append(issues, CustomCommandIssue{
				Index:   index,
				Field:   "description",
				Message: "Telegram custom command is missing a description.",
			})
			continue
		}
		if utf8.RuneCountInString(strings.TrimSpace(entry.Description)) > telegramCommandDescriptionMax {
			issues = append(issues, CustomCommandIssue{
				Index:   index,
				Field:   "description",
				Message: "Telegram custom command description is too long (max 256 characters); it was truncated.",
			})
		}
		if checkDuplicates {
			seen[normalized] = struct{}{}
		}
		resolved = append(resolved, MenuCommand{
			Command:     normalized,
			Description: description,
		})
	}

	return ResolveCustomCommandsResult{Commands: resolved, Issues: issues}
}

type ResolveCustomCommandsParams struct {
	Commands           []CustomCommandInput
	ReservedCommands   map[string]struct{}
	CheckReserved      bool
	CheckReservedSet   bool
	CheckDuplicates    bool
	CheckDuplicatesSet bool
}

type ResolveCustomCommandsResult struct {
	Commands []MenuCommand
	Issues   []CustomCommandIssue
}

type PluginCommandSpec struct {
	Name        string
	Description string
}

func BuildPluginTelegramMenuCommands(params BuildPluginCommandsParams) (BuildPluginCommandsResult, error) {
	specs := params.Specs
	existing := params.ExistingCommands
	if existing == nil {
		existing = make(map[string]struct{})
	}
	commands := make([]MenuCommand, 0, len(specs))
	issues := make([]string, 0)
	pluginCommandNames := make(map[string]struct{})

	for _, spec := range specs {
		normalized := NormalizeTelegramCommandName(spec.Name)
		if normalized == "" || !telegramCommandNamePattern.MatchString(normalized) {
			issues = append(issues, "Plugin command is invalid for Telegram (use a-z, 0-9, underscore; max 32 chars).")
			continue
		}
		description := strings.TrimSpace(spec.Description)
		if description == "" {
			issues = append(issues, "Plugin command is missing a description.")
			continue
		}
		if utf8.RuneCountInString(description) > telegramCommandDescriptionMax {
			description = truncateTelegramDescription(description, telegramCommandDescriptionMax)
			issues = append(issues, "Plugin command description is too long (max 256 characters); it was truncated.")
		}
		if _, ok := existing[normalized]; ok {
			if _, dup := pluginCommandNames[normalized]; dup {
				issues = append(issues, "Plugin command is duplicated.")
			} else {
				issues = append(issues, "Plugin command conflicts with an existing Telegram command.")
			}
			continue
		}
		pluginCommandNames[normalized] = struct{}{}
		existing[normalized] = struct{}{}
		commands = append(commands, MenuCommand{Command: normalized, Description: description})
	}

	return BuildPluginCommandsResult{Commands: commands, Issues: issues}, nil
}

type BuildPluginCommandsParams struct {
	Specs            []PluginCommandSpec
	ExistingCommands map[string]struct{}
}

type BuildPluginCommandsResult struct {
	Commands []MenuCommand
	Issues   []string
}

func BuildCappedTelegramMenuCommands(params BuildCappedCommandsParams) BuildCappedCommandsResult {
	maxCommands := params.MaxCommands
	if maxCommands <= 0 {
		maxCommands = TelegramMaxCommands
	}
	totalCommands := len(params.AllCommands)
	overflowCount := totalCommands - maxCommands
	if overflowCount < 0 {
		overflowCount = 0
	}
	commandsToRegister := params.AllCommands
	if len(commandsToRegister) > maxCommands {
		commandsToRegister = params.AllCommands[:maxCommands]
	}
	return BuildCappedCommandsResult{
		CommandsToRegister: commandsToRegister,
		TotalCommands:      totalCommands,
		MaxCommands:        maxCommands,
		OverflowCount:      overflowCount,
	}
}

type BuildCappedCommandsParams struct {
	AllCommands []MenuCommand
	MaxCommands int
}

type BuildCappedCommandsResult struct {
	CommandsToRegister []MenuCommand
	TotalCommands      int
	MaxCommands        int
	OverflowCount      int
}
