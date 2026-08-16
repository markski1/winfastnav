package presentation

import "sync"

type Page uint8

const (
	PageLauncher Page = iota
	PageMenu
	PageHelp
	PageSettings
	PageAbout
)

type CommandKind uint8

const (
	CommandShow CommandKind = iota
	CommandHide
	CommandSetMode
	CommandSetQuery
	CommandSetMessage
	CommandSetLoading
	CommandSetPage
	CommandSetResults
	CommandSelectResult
	CommandFocusSearch
	CommandFocusHandled
)

type Command struct {
	Kind        CommandKind
	Mode        int
	Query       string
	Message     string
	Loading     bool
	Page        Page
	ResultCount int
	Selected    int
}

type State struct {
	Visible     bool
	Mode        int
	Query       string
	Message     string
	Loading     bool
	Page        Page
	ResultCount int
	Selected    int
	FocusSearch bool
}

type queuedCommand struct {
	command  Command
	response chan State
}

type Controller struct {
	mu         sync.RWMutex
	state      State
	commands   chan queuedCommand
	done       chan struct{}
	stopOnce   sync.Once
	invalidate func()
}

func NewController(initialMode int) *Controller {
	c := &Controller{
		state: State{
			Mode:     initialMode,
			Page:     PageLauncher,
			Selected: -1,
		},
		commands: make(chan queuedCommand, 32),
		done:     make(chan struct{}),
	}
	go c.run()
	return c
}

func (c *Controller) Close() {
	c.stopOnce.Do(func() {
		close(c.done)
	})
}

func (c *Controller) Post(command Command) bool {
	return c.enqueue(queuedCommand{command: command})
}

func (c *Controller) Dispatch(command Command) (State, bool) {
	response := make(chan State, 1)
	if !c.enqueue(queuedCommand{command: command, response: response}) {
		return State{}, false
	}

	select {
	case state := <-response:
		return state, true
	case <-c.done:
		return State{}, false
	}
}

func (c *Controller) Snapshot() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Controller) SetInvalidator(invalidate func()) {
	c.mu.Lock()
	c.invalidate = invalidate
	c.mu.Unlock()
}

func (c *Controller) enqueue(command queuedCommand) bool {
	select {
	case <-c.done:
		return false
	case c.commands <- command:
		return true
	}
}

func (c *Controller) run() {
	for {
		select {
		case <-c.done:
			return
		case queued := <-c.commands:
			state, invalidate := c.apply(queued.command)
			if queued.response != nil {
				queued.response <- state
			}
			if invalidate != nil {
				invalidate()
			}
		}
	}
}

func (c *Controller) apply(command Command) (State, func()) {
	c.mu.Lock()
	switch command.Kind {
	case CommandShow:
		c.state.Visible = true
	case CommandHide:
		c.state.Visible = false
	case CommandSetMode:
		c.state.Mode = command.Mode
		c.state.Selected = -1
	case CommandSetQuery:
		c.state.Query = command.Query
		c.state.Selected = -1
	case CommandSetMessage:
		c.state.Message = command.Message
	case CommandSetLoading:
		c.state.Loading = command.Loading
	case CommandSetPage:
		c.state.Page = command.Page
		c.state.FocusSearch = command.Page == PageLauncher
	case CommandSetResults:
		c.state.ResultCount = max(command.ResultCount, 0)
		c.state.Selected = -1
	case CommandSelectResult:
		if command.Selected >= 0 && command.Selected < c.state.ResultCount {
			c.state.Selected = command.Selected
		} else {
			c.state.Selected = -1
		}
	case CommandFocusSearch:
		c.state.FocusSearch = true
	case CommandFocusHandled:
		c.state.FocusSearch = false
	}
	state := c.state
	invalidate := c.invalidate
	c.mu.Unlock()
	return state, invalidate
}
