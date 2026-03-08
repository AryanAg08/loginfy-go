package core

type Loginfy struct {
	strategies map[string]Strategy
	storage    Storage
	session    SessionManager
	hooks      Hooks
}

func New() *Loginfy {
	return &Loginfy{
		strategies: make(map[string]Strategy),
	}
}

func (l *Loginfy) Use(strategy Strategy) {
	l.strategies[strategy.Name()] = strategy
}
