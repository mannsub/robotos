package behavior

// State represents the behavior service state.
// Mapped from NeoDM DecisionResponse.action.
type State string

const (
	StateIdle       State = "IDLE"
	StateNavigating State = "NAVIGATING"
	StateStopped    State = "STOPPED"
	StateDocking    State = "DOCKING"
)

// Event triggers a state transition.
type Event string

const (
	EventNavigate Event = "NAVIGATE"
	EventIdle     Event = "IDLE"
	EventStop     Event = "STOP"
	EventDock     Event = "DOCK"
	EventRecover  Event = "RECOVER"
)

var transition = map[State]map[Event]State{
	StateIdle: {
		EventNavigate: StateNavigating,
		EventStop:     StateStopped,
		EventDock:     StateDocking,
	},
	StateNavigating: {
		EventIdle: StateIdle,
		EventStop: StateStopped,
		EventDock: StateDocking,
	},
	StateStopped: {
		EventRecover: StateIdle,
	},
	StateDocking: {
		EventIdle: StateIdle,
		EventStop: StateStopped,
	},
}

func nextState(current State, event Event) State {
	if next, ok := transition[current][event]; ok {
		return next
	}
	return current
}
