package main

import (
	"fmt"
	"os"
	"sync/atomic"
)

func startKeyListener() {
	go func() {
		buf := make([]byte, 1)
		for {
			select {
			case <-exitCh:
				return
			default:
				_, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
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
					signalExit()
					return
				}
			}
		}
	}()
}
