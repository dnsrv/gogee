package logging

//goland:noinspection GoUnusedExportedFunction
func NewLoggerMock() *LoggerMock {
	return &LoggerMock{
		CallsCount:   0,
		Messages:     make(map[int]string),
		MessageTypes: make(map[int]string),
	}
}

type LoggerMock struct {
	Prefix       string
	Messages     map[int]string
	MessageTypes map[int]string
	CallsCount   int
}

func (o *LoggerMock) Info(text string) {
	o.CallsCount++
	o.MessageTypes[o.CallsCount] = "info"
	o.Messages[o.CallsCount] = text
}

func (o *LoggerMock) Error(text string) {
	o.CallsCount++
	o.MessageTypes[o.CallsCount] = "warn"
	o.Messages[o.CallsCount] = text
}

func (o *LoggerMock) Fatal(text string) {
	o.CallsCount++
	o.MessageTypes[o.CallsCount] = "fatal"
	o.Messages[o.CallsCount] = text
}

func (o *LoggerMock) WithPrefix(p string) LoggerInterface {
	return &LoggerMock{
		Prefix: p,
	}
}

func (o *LoggerMock) Close() {}

func (o *LoggerMock) GetLastMessageText() string {
	return o.Messages[o.CallsCount]
}
