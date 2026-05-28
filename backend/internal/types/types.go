package types

type GameState string

const (
	StateLobby    GameState = "LOBBY"
	StateChoosing GameState = "CHOOSING"
	StatePlaying  GameState = "PLAYING"
	StateFinished GameState = "FINISHED"
)

type RoundType string

const (
	RoundNone   RoundType = "NONE"
	RoundCifras RoundType = "CIFRAS"
	RoundLetras RoundType = "LETRAS"
)

type Player struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Score   int    `json:"score"`
	IsReady bool   `json:"isReady"`
}

type PlayerResult struct {
	PlayerID    string `json:"playerId,omitempty"`
	Name        string `json:"name"`
	FinalNumber int    `json:"finalNumber"`
	Distance    int    `json:"distance"`
	Expression  string `json:"expression,omitempty"`
	Word        string `json:"word,omitempty"`
	Points      int    `json:"points"`
}

type SyncData struct {
	State              GameState      `json:"state"`
	RoundType          RoundType      `json:"roundType,omitempty"`
	CurrentRound       int            `json:"currentRound"`
	Players            []Player       `json:"players"`
	Chooser            string         `json:"chooser,omitempty"`
	ChooserID          string         `json:"chooserId,omitempty"`
	TargetNumber       int            `json:"targetNumber,omitempty"`
	Numbers            []int          `json:"numbers,omitempty"`
	Letters            []string       `json:"letters,omitempty"`
	EndTime            int64          `json:"endTime,omitempty"`
	ServerTime         int64          `json:"serverTime,omitempty"`
	Winner             string         `json:"winner,omitempty"`
	Solution           string         `json:"solution,omitempty"`
	SolutionSteps      []string       `json:"solutionSteps,omitempty"`
	ExactSolutionSteps []string       `json:"exactSolutionSteps,omitempty"`
	OtherResults       []PlayerResult `json:"otherResults,omitempty"`
}

type ServerMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type ToastMessage struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type ClientMessage struct {
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Vowels int    `json:"vowels,omitempty"`
	Word   string `json:"word,omitempty"`
	Number int    `json:"number,omitempty"`
	Expr   string `json:"expr,omitempty"`
}
