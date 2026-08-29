package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Test application started!")
	i := 0
	for {
		fmt.Printf("Tick %d at %s\n", i, time.Now().Format("15:04:05"))
		i++
		time.Sleep(2 * time.Second)
	}
}
