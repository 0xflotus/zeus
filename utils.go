/*
 *  ZEUS - A Powerful Build System
 *  Copyright (c) 2017 Philipp Mieden <dreadl0ck@protonmail.ch>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/mgutz/ansi"
)

var (

	// color all output to Stderr red
	cWriter = newColorWriter(os.Stderr, ansi.Red)

	// prompt for the interactive
	zeusPrompt  = "zeus"
	signalMutex = &sync.Mutex{}
)

// create a new color writer instance
func newColorWriter(w io.Writer, color string) *colorWriter {
	return &colorWriter{
		color: color,
		w:     w,
	}
}

// colorWriter wraps an io.Writer and writes all data prefixed with specified ANSI string
type colorWriter struct {
	color string
	w     io.Writer
}

// implement io.Writer
func (c *colorWriter) Write(b []byte) (n int, err error) {

	var coloredBuffer = append([]byte(c.color), b...)

	_, err = c.w.Write(append(coloredBuffer, []byte(ansi.Reset)...))
	if err != nil {
		Log.WithError(err).Error("error writing")
	}

	// we need to lie about the written bytelength, otherwise a runtime error will happen
	return len(b), err
}

// dump the currently executed script in case of an error
func dumpScript(script string) {

	f, err := os.OpenFile("error_dump.sh", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		Log.WithError(err).Error("failed to open dump file")
		return
	}
	defer f.Close()

	f.WriteString(script)
	Log.Info("script dumped")
}

// print the current script to stdout
func printScript(script string) {

	fmt.Println(" |---------------------------------------------------------------------------------------------|")
	fmt.Println("     Script")
	fmt.Println(" |---------------------------------------------------------------------------------------------|")
	for i, s := range strings.Split(script, "\n") {

		var lineNumber string
		switch true {
		case i > 9:
			lineNumber = strconv.Itoa(i) + " "
		case i > 99:
			lineNumber = strconv.Itoa(i)
		default:
			lineNumber = strconv.Itoa(i) + "  "
		}
		fmt.Println(" "+lineNumber, s)
	}
	fmt.Println(" |---------------------------------------------------------------------------------------------|")
}

// handle OS SIGNALS for a clean exit and clean up all spawned processes
func handleSignals() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGSEGV, syscall.SIGHUP, syscall.SIGQUIT)

	// var signalLock sync.Mutex
	go func() {

		sig := <-c

		Log.Info("received SIGNAL: ", sig)

		// lock the mutex
		signalMutex.Lock()

		// kill all spawned procs
		clearProcessMap(sig)

		// return to interactive shell
		return
	}()
}

// clean up the mess when we leave
func clearProcessMap(sig os.Signal) {

	// l.Println("processMap:", processMap)

	// range processes
	for name, p := range processMap {
		if p != nil {

			l.Println(printPrompt() + "killing " + name)

			// kill it
			var err error
			if sig == nil {
				err = p.Kill()
			} else {
				err = p.Signal(sig)
			}
			if err != nil {
				Log.WithError(err).Debug("failed to kill " + name)
			}
		}
	}
}

// pad the input string up to the given number of space characters
func pad(in string, length int) string {
	if len(in) < length {
		return fmt.Sprintf("%-"+strconv.Itoa(length)+"s", in)
	}
	return in
}

// create a readable string from commandChain commands
// example: (clean -> build -> install)
func formatcommandChain(commands commandChain) string {

	var out = "("
	for i, cmd := range commands {

		out += cmd.name

		// check if command has params set
		if len(cmd.params) > 0 {
			for _, p := range cmd.params {
				out += " " + p
			}
		}

		// if not last elem
		if !(i == len(commands)-1) {
			out += " -> "
		}
	}
	if out == "(" {
		return ""
	}
	return out + ")"
}

// ClearScreen prints ANSI escape to flush screen
func clearScreen() {
	print("\033[H\033[2J")
}

// count total length of a commandchain
func countCommandChain(chain commandChain) int {
	count := 0
	for _, cmd := range chain {
		count++
		if len(cmd.commandChain) > 0 {
			count += countCommandChain(cmd.commandChain)
		}
	}
	return count
}

func getTotalCommandCount(c *command) int {
	return 1 + countCommandChain(c.commandChain)
}

// print the prompt for the interactive shell
func printPrompt() string {
	// return color.New(color.FgHiRed, color.Bold).SprintFunc()(zeusPrompt + " » ")
	return cp.colorPrompt + zeusPrompt + " » " + cp.colorText
}

// pass the command to the underlying shell
// arguments that contain string literals " or ' will be grouped before passing them to shell
func passCommandToShell(commandName string, args []string) error {

	// handle string literals
	args = fixArgs(args)

	var cmd *exec.Cmd

	// if there are arguments pass them
	if len(args) > 0 {
		cmd = exec.Command(commandName, args...)
	} else {
		cmd = exec.Command(commandName)
	}

	// setup environment
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// fix args in case there is a string literal in there
// this will cause all strings in arguments to be passed as one to the shell
// example:
// ["git", "commit", "-m", "'what", "the", "hell'"] -> ["git", "commit", "-m", "'what the hell'"]
func fixArgs(args []string) []string {

	var (
		fixed         = []string{}
		insideLiteral bool
		literalIndex  int
	)

	// range arguments until there appears a starting string literal
	// from there on concatenate all following fields to the current one
	// when the closing tag appears concatenation is stopped
	for _, a := range args {

		if insideLiteral {
			fixed[literalIndex] += " " + a
		} else {
			fixed = append(fixed, a)
		}

		if isStartTag(a) {
			insideLiteral = true
			literalIndex = len(fixed) - 1
		} else if isEndTag(a) {
			insideLiteral = false
		}
	}

	return fixed
}

// check if the string literal starts
func isStartTag(s string) bool {
	if strings.HasPrefix(s, "\"") || strings.HasPrefix(s, "'") {
		return true
	}
	return false
}

// check if the string literal ends
func isEndTag(s string) bool {
	if strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'") {
		return true
	}
	return false
}

// check if its a valid command chain
func validCommandChain(args []string) bool {

	var (
		chain       = strings.Join(args, " ")
		job         = p.AddJob(chain)
		commandList = parseCommandChain(chain)
	)

	defer p.RemoveJob(job)

	_, err := job.getCommandChain(commandList)
	if err != nil {
		Log.WithError(err).Error("failed to get command chain")
		return false
	}

	return true
}

// handle help shell command
func handleHelpCommand(args []string) {

	if len(args) < 2 {
		printHelpUsageErr()
		return
	}

	if c, ok := commands[args[1]]; ok {
		l.Println("\n" + c.manual)
		return
	}

	printHelpUsageErr()
}

func printHelpUsageErr() {
	Log.Error(ErrInvalidUsage)
	Log.Info("usage: help <command>")
}

// check if the argument type matches the expected one
func validArgType(in string, k reflect.Kind) bool {

	var err error

	switch k {
	case reflect.Bool:
		_, err = strconv.ParseBool(in)
	case reflect.Float64:
		_, err = strconv.ParseFloat(in, 64)
	case reflect.String:
	case reflect.Int:
		_, err = strconv.ParseInt(in, 64, 0)
	default:
		return false
	}

	if err == nil {
		return true
	}
	return false
}
