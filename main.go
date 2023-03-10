package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/google/shlex"
	"golang.org/x/sync/errgroup"
)

func main() {
	if os.Getenv("PPROF") != "" {
		filename := fmt.Sprintf("sprunner-pprof-%s.pprof", time.Now().Format(time.RFC3339))
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	if err := mainImpl(os.Args[1:]); err != nil {
		fmt.Println(err.Error())
	}
}

func mainImpl(args []string) error {
	// TODO: handle this for not exactly 2 commands
	if len(args) != 2 {
		return errors.New("can only tile exactly 2 commands")
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	commands := make([]*exec.Cmd, len(args))
	for i, commandSpec := range args {
		command, err := cmdFromSpec(commandSpec)
		if err != nil {
			return err
		}
		commands[i] = command
	}

	group := errgroup.Group{}
	for i, command := range commands {
		subscreen := NewSubscreen(screen, len(commands), i)
		command.Stdout = subscreen
		command.Stderr = subscreen
		group.Go(command.Run)
	}

	running := true
	for running {
		evt := screen.PollEvent()
		switch evt := evt.(type) {
		case *tcell.EventKey:
			if evt.Key() == tcell.KeyCtrlC {
				for _, command := range commands {
					command.Process.Signal(os.Interrupt)
				}
				running = false
			}
		}
	}

	return group.Wait()
}

func cmdFromSpec(commandSpec string) (*exec.Cmd, error) {
	parts, err := shlex.Split(commandSpec)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("cannot build command from 0 parts")
	}

	return exec.Command(parts[0], parts[1:]...), nil
}

type Subscreen struct {
	screen tcell.Screen
	count  int
	index  int
	lines  []string
}

func NewSubscreen(screen tcell.Screen, count int, index int) *Subscreen {
	return &Subscreen{
		screen: screen,
		count:  count,
		index:  index,
		lines:  []string{},
	}
}

// Write implements `io.Writer` and simulates the experience
// of writing to a terminal's stdout or stderr.
func (s *Subscreen) Write(bytes []byte) (n int, err error) {
	startX, startY, endX, endY := s.Subregion()
	width := endX - startX
	height := endY - startY

	// TODO: remove lines at some point :)
	newLines := strings.Split(string(bytes), "\n")
	if len(s.lines) == 0 {
		s.lines = append(s.lines, newLines[0])
	} else {
		s.lines[len(s.lines)-1] += newLines[0]
	}
	newLines = newLines[1:]
	s.lines = append(s.lines, newLines...)

	if len(s.lines) > height {
		s.lines = s.lines[len(s.lines)-height-1:]
	}

	for row, line := range s.lines {
		for col, r := range line {
			if col >= width {
				break
			}
			s.screen.SetContent(startX+col, startY+row, r, []rune{}, tcell.StyleDefault)
		}
	}
	s.screen.Show()
	return len(bytes), nil
}

// Subregion returns the top-left and bottom-right corners as (x, y) coordinates
// of this subscreen.
func (s *Subscreen) Subregion() (int, int, int, int) {
	// TODO: this assumes that we only have two possible screens (true for now)
	// and so it over-simplifies this sub-region calculation.
	// whenever we generalize to more regions, make this more complicated!
	width, height := s.screen.Size()
	middle := width / 2

	var xStart, xEnd int
	if s.index == 0 {
		xStart = 0
		xEnd = middle
	} else if s.index == 1 {
		xStart = middle
		xEnd = width
	} else {
		panic("Can only handle 2 screens")
	}

	return xStart, 0, xEnd, height
}

func renderString(screen tcell.Screen, s string) {
	for i, r := range s {
		screen.SetContent(i, 0, r, []rune{}, tcell.StyleDefault)
	}
	screen.Show()
}
