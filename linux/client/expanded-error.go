package main

import (
	"fmt"
)

type Error struct {
	err    error
	source string
}

func (err Error) Error() string {
	return err.String()
}

func (err Error) String() string {
	return fmt.Sprintf("%v: %v", err.source, err.err)
}
