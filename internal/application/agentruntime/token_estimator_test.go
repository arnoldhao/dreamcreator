package agentruntime

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestEstimateMessageTokens_StripsToolResultDetails(t *testing.T) {
	t.Parallel()

	base := schema.ToolMessage(`{"result":"ok"}`, "call-1")
	withDetails := schema.ToolMessage(`{"result":"ok","details":{"payload":"`+strings.Repeat("x", 5000)+`"}}`, "call-1")

	baseTokens := EstimateMessageTokens(base)
	withDetailsTokens := EstimateMessageTokens(withDetails)
	if baseTokens != withDetailsTokens {
		t.Fatalf("expected details to be ignored for token estimate, base=%d withDetails=%d", baseTokens, withDetailsTokens)
	}
}

func TestEstimateMessageTokensSafe_AppliesSafetyMargin(t *testing.T) {
	t.Parallel()

	message := &schema.Message{
		Role:    schema.User,
		Content: "hello world, this is a token estimator safety margin test",
	}
	raw := EstimateMessageTokens(message)
	safe := EstimateMessageTokensSafe(message)
	if safe <= raw {
		t.Fatalf("expected safe estimate to be larger than raw estimate, raw=%d safe=%d", raw, safe)
	}
}
