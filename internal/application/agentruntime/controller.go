package agentruntime

import (
	"context"
	"strings"
	"sync"
	"time"
)

// AgentController implements the runtime control plane:
// steer, followUp, abort, waitForIdle, retry, timeout.
type AgentController struct {
	mu        sync.Mutex
	steerQ    []string
	followQ   []string
	inFlight  int
	idleCh    chan struct{}
	aborted   bool
	abortText string
	retry     int
	timeout   time.Duration
}

func NewAgentController() *AgentController {
	idle := make(chan struct{})
	close(idle)
	return &AgentController{
		idleCh: idle,
	}
}

func (c *AgentController) BeginRun() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight++
	if c.inFlight == 1 {
		c.idleCh = make(chan struct{})
	}
}

func (c *AgentController) EndRun() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inFlight > 0 {
		c.inFlight--
	}
	if c.inFlight == 0 {
		close(c.idleCh)
	}
}

func (c *AgentController) Steer(message string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}
	c.mu.Lock()
	c.steerQ = append(c.steerQ, trimmed)
	c.mu.Unlock()
}

func (c *AgentController) FollowUp(message string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}
	c.mu.Lock()
	c.followQ = append(c.followQ, trimmed)
	c.mu.Unlock()
}

func (c *AgentController) NextSteer() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.steerQ) == 0 {
		return "", false
	}
	value := c.steerQ[0]
	c.steerQ = c.steerQ[1:]
	return value, true
}

func (c *AgentController) NextFollowUp() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.followQ) == 0 {
		return "", false
	}
	value := c.followQ[0]
	c.followQ = c.followQ[1:]
	return value, true
}

func (c *AgentController) Abort(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aborted = true
	c.abortText = strings.TrimSpace(reason)
}

func (c *AgentController) ResetAbort() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aborted = false
	c.abortText = ""
}

func (c *AgentController) Aborted() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.aborted {
		return "", false
	}
	return c.abortText, true
}

func (c *AgentController) WaitForIdle(ctx context.Context) error {
	c.mu.Lock()
	idle := c.idleCh
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-idle:
		return nil
	}
}

func (c *AgentController) Retry() {
	c.mu.Lock()
	c.retry++
	c.mu.Unlock()
}

func (c *AgentController) RetryCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.retry
}

func (c *AgentController) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	c.timeout = timeout
	c.mu.Unlock()
}

func (c *AgentController) Timeout() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.timeout
}
