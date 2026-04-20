package runtime

import (
	"strings"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/agentruntime"
)

type promptContextBuildConfig struct {
	contextWindowTokens int
	reserveTokens       int
	extraTokens         int
	systemPrompt        string
}

func buildPromptMessagesToBudget(messages []*schema.Message, config promptContextBuildConfig) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	if config.contextWindowTokens <= 0 {
		return messages
	}
	budget := resolvePromptMessageBudget(config)
	if budget <= 0 {
		return keepRecentMessagesByBudget(messages, 1)
	}
	if agentruntime.EstimateMessagesTokensSafe(messages) <= budget {
		return messages
	}

	systemMessages, summary, nonSystem := splitSystemAndSummary(messages)
	lockedSystem := cloneSchemaMessages(systemMessages)
	if strings.TrimSpace(summary) != "" {
		lockedSystem = append(lockedSystem, &schema.Message{
			Role:    schema.System,
			Content: compactionSummaryPrefix + strings.TrimSpace(summary),
		})
	}
	if len(nonSystem) == 0 {
		if agentruntime.EstimateMessagesTokensSafe(lockedSystem) <= budget {
			return lockedSystem
		}
		return nil
	}

	recent := fitPromptNonSystemMessages(nonSystem, budget, lockedSystem)
	if len(recent) == 0 {
		recent = keepRecentMessagesByBudget(nonSystem, 1)
	}
	combined := append(cloneSchemaMessages(lockedSystem), recent...)
	if len(lockedSystem) > 0 && agentruntime.EstimateMessagesTokensSafe(combined) > budget {
		return fitPromptNonSystemMessages(nonSystem, budget, nil)
	}
	return combined
}

func estimatePromptMessagesTokens(messages []*schema.Message) int {
	return agentruntime.EstimateMessagesTokensSafe(messages)
}

func resolvePromptMessageBudget(config promptContextBuildConfig) int {
	budget := config.contextWindowTokens
	if budget <= 0 {
		return 0
	}
	if config.reserveTokens > 0 {
		budget -= config.reserveTokens
	}
	if config.extraTokens > 0 {
		budget -= config.extraTokens
	}
	if systemPrompt := strings.TrimSpace(config.systemPrompt); systemPrompt != "" {
		budget -= agentruntime.EstimateMessageTokensSafe(&schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
	}
	if budget < 0 {
		return 0
	}
	return budget
}

func fitPromptNonSystemMessages(messages []*schema.Message, budget int, lockedSystem []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	available := budget
	if len(lockedSystem) > 0 {
		available -= agentruntime.EstimateMessagesTokensSafe(lockedSystem)
	}
	if available <= 0 {
		return nil
	}
	return keepRecentMessagesByBudget(messages, available)
}

func keepRecentMessagesByBudget(messages []*schema.Message, budget int) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	if budget <= 0 {
		last := messages[len(messages)-1]
		if last == nil {
			return nil
		}
		return []*schema.Message{last}
	}
	kept := make([]*schema.Message, 0, len(messages))
	remaining := budget
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message == nil {
			continue
		}
		tokens := agentruntime.EstimateMessageTokensSafe(message)
		if len(kept) > 0 && remaining-tokens < 0 {
			break
		}
		kept = append(kept, message)
		remaining -= tokens
	}
	if len(kept) == 0 {
		last := messages[len(messages)-1]
		if last == nil {
			return nil
		}
		return []*schema.Message{last}
	}
	reverseMessages(kept)
	return kept
}
