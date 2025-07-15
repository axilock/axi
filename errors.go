package main

import (
	"fmt"
	runtimeDebug "runtime/debug"
)

type RuntimePanic struct {
	trace string
}

func NewRuntimePanic(p any) *RuntimePanic {
	err, ok := p.(error)
	errMessage := ""
	if ok {
		errMessage = err.Error()
	} else {
		errMessage = fmt.Sprintf("%v", p)
	}
	return &RuntimePanic{trace: errMessage + ":" + string(runtimeDebug.Stack())}
}

func (r *RuntimePanic) Error() string {
	return r.trace
}
