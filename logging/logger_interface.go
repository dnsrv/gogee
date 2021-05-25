package logging

const (
	LogLevelPrefixInfo  = "[info]"
	LogLevelPrefixWarn  = "[warn]"
	LogLevelPrefixFatal = "[fatal]"
)

type LoggerInterface interface {
	Info(string)
	Error(string)
	Fatal(string)
	WithPrefix(string) LoggerInterface
	Close()
}
