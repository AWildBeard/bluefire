package main

import (
	"fmt"
)

// Error allows us to more granuarly control asynchronous errors and perform actions
// based on those errors. Error specifically is used for the scan functionality of
// this program
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
