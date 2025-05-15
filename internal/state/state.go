package state

// Direction specifies navigation Direction in logs
type Direction int

const (
	Next Direction = iota
	Home
	Prev
)

type UIState struct {
	LogGroup  *LogGroup
	LogStream *LogStream
	LogEvent  *LogEvent
}

func NewUIState() *UIState {
	return &UIState{
		LogEvent: &LogEvent{
			enableOutputFile: false,
		},
		LogGroup: &LogGroup{
			pageTokens: make(map[int]*string),
		},
		LogStream: &LogStream{
			pageTokens: make(map[int]*string),
		},
	}
}
