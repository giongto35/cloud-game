package turnc

// TODO(ar): Move to logging package.
type nopLogger struct{}

func (nopLogger) Trace(msg string)                          {}
func (nopLogger) Tracef(format string, args ...interface{}) {}
func (nopLogger) Debug(msg string)                          {}
func (nopLogger) Debugf(format string, args ...interface{}) {}
func (nopLogger) Info(msg string)                           {}
func (nopLogger) Infof(format string, args ...interface{})  {}
func (nopLogger) Warn(msg string)                           {}
func (nopLogger) Warnf(format string, args ...interface{})  {}
func (nopLogger) Error(msg string)                          {}
func (nopLogger) Errorf(format string, args ...interface{}) {}
