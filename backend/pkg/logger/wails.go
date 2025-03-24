package logger

// WailsLogger 实现 wails.Logger 接口的适配器
type WailsLogger struct{}

func NewWailsLogger() *WailsLogger {
	return &WailsLogger{}
}

func (l *WailsLogger) Print(message string) {
	Info(message)
}

func (l *WailsLogger) Trace(message string) {
	Debug(message)
}

func (l *WailsLogger) Debug(message string) {
	Debug(message)
}

func (l *WailsLogger) Info(message string) {
	Info(message)
}

func (l *WailsLogger) Warning(message string) {
	Warn(message)
}

func (l *WailsLogger) Error(message string) {
	Error(message)
}

func (l *WailsLogger) Fatal(message string) {
	Error("FATAL: " + message)
}
