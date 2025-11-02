package main

import (
	"fmt"
	"golang.org/x/term"
	"os"
	"sync/atomic"
)

func startKeyListener() {
	go func() {
		fd := int(os.Stdin.Fd())
    oldState, err := term.MakeRaw(fd)
		if err != nil {
			panic(err)
		}
		defer term.Restore(fd, oldState)

		buf := make([]byte, 1)
		for {
			os.Stdin.Read(buf)
			switch buf[0] {
			case ' ':
				atomic.StoreInt32(&paused, 1-atomic.LoadInt32(&paused))
				if atomic.LoadInt32(&paused) == 1 {
					fmt.Print("\n\r⏸ paused")
				} else {
					fmt.Print("\n\rUnpaused")
				}
			case 'e', 'E':
				fmt.Println("\n\rExiting...")
				close(exitCh)
				return
			}
		}
	}()
}

