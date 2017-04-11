package main

import (
	"fmt"
	"syscall"
	"time"
)

func main() {
	tv := new(syscall.Timeval)
	syscall.Gettimeofday(tv)
	fmt.Println(time.Unix(tv.Sec, tv.Usec))
	fmt.Println("It's alive!")
}
