package proc

type State int

const (
	StateIdle State = iota
	StateRunning
	StateError
	StateSuccess
)

var stateName = map[State]string{
	StateIdle:    "idle",
	StateRunning: "running",
	StateError:   "error",
	StateSuccess: "success",
}

func (as State) String() string {
	return stateName[as]
}
