package environment

type Env string

const (
	Dev        Env = "dev"
	Staging        = "staging"
	Production     = "prod"
)

func (env *Env) AnyOf(what ...Env) bool {
	for _, cur := range what {
		if *env == cur {
			return true
		}
	}
	return false
}
