package checker

// Env は変数名から型へのマップ。outer で外側スコープを参照する
type Env struct {
	store map[string]*Type
	outer *Env
}

func NewEnv() *Env {
	return &Env{
		store: make(map[string]*Type),
	}
}

func NewEnclosedEnvironment(outer *Env) *Env {
	env := NewEnv()
	env.outer = outer
	return env
}

// Get は現在スコープから親へたどって最初に見つかった型を返す
func (e *Env) Get(name string) (*Type, bool) {
	if t, ok := e.store[name]; ok {
		return t, true
	}
	if e.outer != nil {
		return e.outer.Get(name)
	}
	return nil, false
}

// Set は現在のスコープに宣言する
func (e *Env) Set(name string, t *Type) {
	e.store[name] = t
}
