package object

type BuiltinFunction func(args ...Object) (Object, error)

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType {
	return "BUILTIN"
}

func (b *Builtin) Inspect() string {
	return "builtin function"
}
