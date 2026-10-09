package checker

type Env struct {
	store map[string]Type
}

func NewEnv() *Env {
	return &Env{
		store: make(map[string]Type),
	}
}

func (e *Env) Get(name string) (Type, bool) {
	t, ok := e.store[name]
	return t, ok
}

func (e *Env) Set(name string, t Type) {
	e.store[name] = t
}
