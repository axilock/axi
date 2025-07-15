package main

import "time"

type SleepCmd struct {
	Duration int `arg:"" help:"Seconds to sleep"`
}

func (s *SleepCmd) Run() error {
	time.Sleep(time.Duration(s.Duration) * time.Second)
	return nil
}
