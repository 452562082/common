package utils

import "git.oschina.net/kuaishangtong/common/utils/log"

type ErrLog struct {
}

func (e *ErrLog) Printf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

type InfoLog struct {
}

func (e *InfoLog) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}
