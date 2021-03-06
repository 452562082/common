package utils

import "kuaishangtong/common/utils/log"

type ErrLog struct {
	logger *log.ZeusLogger
}

func NewErrlog() *ErrLog {

	logger := log.NewLogger(100000)
	logger.SetLogFuncCallWithDepth(true, 4)
	logger.SetLogger("console", `{"color":true}`)
	logger.SetLevel(log.LevelDebug)
	return &ErrLog{
		logger: logger,
	}
}

func (e *ErrLog) Printf(format string, v ...interface{}) {
	e.logger.Errorf(format, v...)
}

type InfoLog struct {
	logger *log.ZeusLogger
}

func NewInfoLog() *InfoLog {
	logger := log.NewLogger(100000)
	logger.SetLogFuncCallWithDepth(true, 4)
	logger.SetLogger("console", `{"color":true}`)
	logger.SetLevel(log.LevelDebug)
	return &InfoLog{
		logger: logger,
	}
}

func (e *InfoLog) Printf(format string, v ...interface{}) {
	e.logger.Infof(format, v...)
}
