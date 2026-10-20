package main

import (
	"fmt"
	"os"
	"sync/atomic"
)

func startKeyListener() {
	go func() {
		buf := make([]byte, 3)
		for {
			select {
			case <-exitCh:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
				if n == 3 && buf[0] == 27 && buf[1] == 91 {
					switch buf[2] {
					case 65:
						interval := adjustTickInterval(-tickIntervalStep)
						fmt.Printf("\n\rSpeed: %dms/tick", interval.Milliseconds())
					case 66:
						interval := adjustTickInterval(tickIntervalStep)
						fmt.Printf("\n\rSpeed: %dms/tick", interval.Milliseconds())
					}
					continue
				}
				switch buf[0] {
				case 'r', 'R':
					atomic.StoreInt32(&currentReportView, int32(reportViewReproduction))
				case '?':
					atomic.StoreInt32(&currentReportView, int32(reportViewHelp))
				case ' ':
					atomic.StoreInt32(&paused, 1-atomic.LoadInt32(&paused))
					if atomic.LoadInt32(&paused) == 1 {
						fmt.Print("\n\r⏸ paused")
					} else {
						fmt.Print("\n\rUnpaused")
					}
				case 'c', 'C':
					select {
					case spawnDietCh <- DietCarnivore:
					default:
					}
					fmt.Print("\n\rQueued spawn: Carnivore")
				case 'h', 'H':
					select {
					case spawnDietCh <- DietHerbivore:
					default:
					}
					fmt.Print("\n\rQueued spawn: Herbivore")
				case 'e', 'E':
					fmt.Println("\n\rExiting...")
					signalExit()
					return
				}
			}
		}
	}()
}
