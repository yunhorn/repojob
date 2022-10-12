package main

import (
	"log"
	"testing"
)

func Test_RemoveMult(t *testing.T) {
	source := []string{}
	source = append(source, "a")
	source = append(source, "b")
	source = append(source, "c")
	dest := []string{}
	dest = append(dest, "c")
	dest = append(dest, "d")
	dest = append(dest, "e")
	result := removeMult(source, dest)
	log.Println(result)
}
