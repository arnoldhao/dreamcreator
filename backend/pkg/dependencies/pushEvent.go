package dependencies

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/types"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type pushEvent struct {
	eventBus events.EventBus
}

func NewPushEvent(eventBus events.EventBus) PushEvent {
	return &pushEvent{
		eventBus: eventBus,
	}
}

// PublishInstallEvent 发布安装事件
func (p *pushEvent) PublishInstallEvent(depType string, stage types.DtTaskStage, percentage float64) {
	event := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicDowntasksInstalling,
		Source:    "dependencies",
		Timestamp: time.Now(),
		Data: &types.DtProgress{
			ID:         fmt.Sprintf("dep-%s", depType),
			Type:       depType,
			Stage:      stage,
			Percentage: percentage,
		},
	}
	p.eventBus.Publish(context.Background(), event)
}
