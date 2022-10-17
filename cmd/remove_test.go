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

func Test_CommandFromComment(t *testing.T) {
	comment := `hello
/assign @user
/kind hello
	`
	ops := CommandFromComment(comment, "user")
	log.Println(len(ops))
	for _, ro := range ops {
		log.Println("ro:", ro.Name, ro.Action, ro.Assigners, len(ro.Assigners))
	}
}
