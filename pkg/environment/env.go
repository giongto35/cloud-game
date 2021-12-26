package environment

type Env string

const (
	Dev        Env = "dev"
	Staging    Env = "staging"
	Production Env = "prod"
)

func (env *Env) AnyOf(what ...Env) bool {
	for _, cur := range what {
		if *env == cur {
			return true
		}
	}
	return false
}
