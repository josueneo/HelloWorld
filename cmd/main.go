package main

import (
	"fmt"
	"hello"
)

func main() {
	message := hello.SayHello("World")
	fmt.Println(message)
}
