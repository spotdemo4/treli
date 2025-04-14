package app

type State int

const (
	StateIdle State = iota
	StateLoading
	StateError
	StateSuccess
)

var stateName = map[State]string{
	StateIdle:    "idle",
	StateLoading: "loading",
	StateError:   "error",
	StateSuccess: "success",
}

func (as State) String() string {
	return stateName[as]
}
