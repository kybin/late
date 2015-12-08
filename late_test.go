package main

import (
	"fmt"
	"testing"
)

// TODO: Make it real test function.
func TestGrouping(t *testing.T) {
	books := scanRootDir()
	fmt.Println(grouping(books))
}

