package nwctl

type ErrConfigValue struct {
	err string
}

func (e *ErrConfigValue) Error() string {
	return e.err
}
