package main

import (
	"flag"
	"fmt"
	"github.com/pkg/term"
	"os"
	"time"
)

type Mode int

const (
	NONE Mode = iota
	TIMER
	COUNTDOWN
	STOPWATCH
)

type Flags struct {
	verbose bool
	quiet   bool
}

var flags = Flags{}

func main() {
	mode, duration := parseFlags()
	if flags.verbose {
		fmt.Printf("Flags: %+v\n", flags)
		fmt.Printf("Mode: %v\n", mode)
		fmt.Printf("Duration: %v\n", duration)
	}
	c := make(chan byte)
	e := make(chan int)

	// put terminal into cbreak mode so we get characters as they are entered
	t, err := term.Open("/dev/tty")
	if err != nil {
		fmt.Printf("Unable to open terminal: %v\n", err)
		os.Exit(1)
	}
	err = t.SetCbreak()
	if err != nil {
		fmt.Printf("Unable to set cbreak mode in terminal: %v\n", err)
		os.Exit(1)
	}
	defer t.Restore()

	go readStdin(c, e)
	ret := runTimer(mode, duration, c, e)
	t.Restore()
	os.Exit(ret)
}

func runTimer(mode Mode, duration time.Duration, c chan byte, e chan int) int {
	start := time.Now()
	if mode == STOPWATCH {
		duration = 1<<63 - 1 // duration is really an int64
	}
	pause := false
	tk := time.NewTicker(time.Millisecond * 10)
	defer tk.Stop()

LOOP:
	for d := time.Since(start); ; {
		select {
		case t := <-tk.C:
			if pause {
				continue
			}
			d = t.Sub(start)
			if d > duration {
				fmt.Print("\a")
				printElapsed(mode, duration, duration)
				break LOOP
			}
			printElapsed(mode, duration, d)
		case char := <-c:
			if char == 'Q' || char == 'q' {
				break LOOP
			}
			if mode == STOPWATCH && char == ' ' {
				if !pause {
					pause = true
				} else {
					start = time.Now().Add(-d)
					pause = false
				}
			}
		case ret := <-e:
			return ret
		}
	}
	fmt.Print("\n")
	return 0
}

func printDuration(duration time.Duration) string {
	hours := duration.Truncate(time.Hour)
	duration = duration - hours
	hours = hours / time.Hour
	minutes := duration.Truncate(time.Minute)
	duration = duration - minutes
	minutes = minutes / time.Minute
	seconds := duration.Truncate(time.Second)
	duration = duration - seconds
	seconds = seconds / time.Second
	milliseconds := duration.Truncate(time.Millisecond) / (time.Millisecond * 10)

	return fmt.Sprintf("[%2.2d:%2.2d:%2.2d.%2.2d]", hours, minutes, seconds, milliseconds)
}

func printElapsed(mode Mode, total time.Duration, duration time.Duration) {
	switch mode {
	case STOPWATCH:
		fallthrough
	case TIMER:
		fmt.Printf("\rElapsed time: %s", printDuration(duration))
	case COUNTDOWN:
		fmt.Printf("\rTime Remaining: %s", printDuration(total-duration))
	}
}

func readStdin(c chan byte, e chan int) {
	b := make([]byte, 1)

	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			fmt.Printf("Error reading stdin: %v\n", err)
			e <- 1
		}
		if flags.verbose {
			fmt.Printf("read %q from stdin\n", b[0])
		}
		// exit if C-d recieved
		if b[0] == '\x04' {
			e <- 0
		}
		c <- b[0]
	}
}

func parseFlags() (Mode, time.Duration) {
	var countdown, timer, stopwatch bool
	var mode Mode

	flag.BoolVar(&flags.verbose, "v", false, "verbose")
	flag.BoolVar(&flags.quiet, "q", false, "quiet")
	flag.BoolVar(&timer, "t", false, "start timer")
	flag.BoolVar(&countdown, "c", false, "start countdown")
	flag.BoolVar(&stopwatch, "s", false, "start stopwatch")

	flag.Parse()

	modes := 0
	if timer {
		mode = TIMER
		modes++
	}
	if countdown {
		mode = COUNTDOWN
		modes++
	}
	if stopwatch {
		mode = STOPWATCH
		modes++
	}
	if modes == 0 {
		fmt.Println("No mode provided")
		os.Exit(1)
	}
	if modes > 1 {
		fmt.Println("Too many modes provided")
		os.Exit(1)
	}

	// TODO: write custom duration parser
	duration, err := time.ParseDuration(flag.Arg(0))
	if err != nil && mode != STOPWATCH {
		fmt.Printf("Parse error: %v\n", err)
		os.Exit(1)
	}

	return mode, duration
}
