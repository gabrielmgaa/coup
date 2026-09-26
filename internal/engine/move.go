package engine

type Move interface{ sealedMove() }

type Act struct {
	By     string
	Action ActionType
	Target string
}

type LoseInfluence struct {
	By   string
	Card Character
}

type Respond struct {
	By        string
	Window    int
	Answer    Answer
	Character Character
}

func (Act) sealedMove()           {}
func (LoseInfluence) sealedMove() {}
func (Respond) sealedMove()       {}
