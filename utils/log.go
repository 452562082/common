package utils

import "kuaishangtong/common/utils/log"

type ErrLog struct {
	logger *log.ZeusLogger
}

func NewErrlog() *ErrLog {

	logger := log.NewLogger(10000)
	logger.SetLogFuncCallDepth(3)
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
	logger := log.NewLogger(10000)
	logger.SetLogFuncCallDepth(3)
	return &InfoLog{
		logger: logger,
	}
}

func (e *InfoLog) Printf(format string, v ...interface{}) {
	e.logger.Infof(format, v...)
}
