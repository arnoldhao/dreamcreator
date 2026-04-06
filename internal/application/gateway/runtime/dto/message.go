package dto

import "dreamcreator/internal/application/chatevent"

type Message struct {
	ID      string                 `json:"id,omitempty"`
	Role    string                 `json:"role"`
	Content string                 `json:"content"`
	Parts   []chatevent.MessagePart `json:"parts,omitempty"`
}
