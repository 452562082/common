package utils

import "kuaishangtong/common/utils/log"

type ErrLog struct {
	logger *log.ZeusLogger
}

func NewErrlog() *ErrLog {
	return &ErrLog{
		log.NewLogger(10000),
	}
}

func (e *ErrLog) Printf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

type InfoLog struct {
	logger *log.ZeusLogger
}

func NewInfoLog() *InfoLog {
	return &InfoLog{
		log.NewLogger(10000),
	}
}

func (e *InfoLog) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}
