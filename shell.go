package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (e *env) read(in *os.File, wc, qc chan<- bool, iq chan<- string) {
	go func() {
		o := true
		for {
			if in == nil {
				in = os.Stdin
			}
			reader := bufio.NewReader(in)
			if o {
				fmt.Print(">>> ")
			} else {
				o = true
			}
			text, err := reader.ReadString('\n')
			if err != nil {
				e.logger("read", "", err)
				qc <- true
			}
			if e.parser.parseLine(text, iq) {
				wc <- true
				o = false
			}
		}
	}()
}

func (e *env) write(bc chan<- bool) {

	go func() {
		f, err := os.OpenFile(e.TmpPath, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return
		}
		time.Sleep(time.Microsecond)

		for _, l := range e.parser.convertLines() {
			f.WriteString(l)
		}
		f.Sync()
		if err := f.Close(); err != nil {
			e.logger("writer", "", err)
			return
		}
		bc <- true
		e.parser.main = []string{}
	}()
}

func (e *env) build(ec chan<- bool) {

	go func() {
		os.Chdir(e.BldDir)
		cmd := "go"
		args := []string{"build", tmpname}
		if err := runCmd(cmd, args...); err != nil {
			e.logger("build", "", err)
			return
		}
		ec <- true
	}()
}

func (e *env) exec(rc chan<- bool) {
	go func() {
		runCmd(strings.Split(filepath.Clean(e.TmpPath), ".")[0], []string{}...)
		rc <- true
	}()
}

func (e *env) shell() {

	qc := make(chan bool)
	rc := make(chan bool)
	wc := make(chan bool)
	bc := make(chan bool)
	ec := make(chan bool)
	iq := make(chan string, 10)

	go e.read(nil, wc, qc, iq)
	go goGet(<-iq)

	for {
		select {
		case <-wc:
			go e.write(bc)
		case <-bc:
			go e.build(ec)
		case <-ec:
			go e.exec(rc)
		}
	}

	time.Sleep(time.Nanosecond)
}
