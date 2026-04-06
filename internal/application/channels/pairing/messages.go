package pairing

import (
	"fmt"
	"strings"
)

const ApprovedMessage = "DreamCreator access approved. Send a message to start chatting."

func BuildPairingReply(channel string, idLine string, code string) string {
	lines := []string{
		"DreamCreator: access not configured.",
		"",
		strings.TrimSpace(idLine),
		"",
		fmt.Sprintf("Pairing code: %s", strings.TrimSpace(code)),
		"",
		"Ask the bot owner to approve in DreamCreator Settings.",
	}
	return strings.Join(lines, "\n")
}
