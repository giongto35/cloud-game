package environment

import "os/user"

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

func GetUserHome() (string, error) {
	me, err := user.Current()
	if err != nil {
		return "", err
	}
	return me.HomeDir, nil
}
