package raft

import (
	"fmt"
)

type A struct {
	i int
}

type B struct {
	A
	j int
}

func main() {

	ch := make(chan bool)

	go func() {
		ch <- true
	}()
	b := <-ch
	fmt.Printf("%b\n", b)
	a := &B{A: A{i: 1}, j: 2}
	fmt.Printf("%d\n", a.i)
}
