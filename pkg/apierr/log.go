package apierr

type Log interface {
	Error(args ...any)
	Errorf(format string, args ...any)
}
