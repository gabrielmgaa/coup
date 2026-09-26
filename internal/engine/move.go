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

type ReturnCards struct {
	By    string
	Cards [2]Character
}

func (Act) sealedMove()           {}
func (ReturnCards) sealedMove()   {}
func (LoseInfluence) sealedMove() {}
func (Respond) sealedMove()       {}
