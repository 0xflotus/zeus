/*
 *  ZEUS - An Electrifying Build System
 *  Copyright (c) 2017 Philipp Mieden <dreadl0ck [at] protonmail [dot] ch>
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
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/mgutz/ansi"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dreadl0ck/readline"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	// current zeus version
	version = "0.7.1"
)

var (
	// Log instance
	Log = logrus.New()

	// all available build target commands
	// does not include the built-ins!
	// command names mapped to command structs
	commands     = make(map[string]*command, 0)
	commandMutex = &sync.Mutex{}

	// readline auto completion
	completer     = newCompleter()
	completerLock = &sync.Mutex{}

	// assets folder
	assetBox = rice.MustFindBox("assets")

	// configuration
	conf *config

	// project data
	projectData *data

	// shell formatter
	f = newFormatter()

	// parser
	p = newParser()

	globalsContent []byte

	debug      bool
	asciiArt   string
	workingDir string

	// total number of commands that will be executed when running the command
	numCommands    int
	currentCommand int

	// running a test?
	testingMode bool
)

func init() {

	// set up formatter
	Log.Formatter = new(prefixed.TextFormatter)
}

func main() {

	var cLog = Log.WithField("prefix", "main")

	if runtime.GOOS == "windows" {
		cLog.Fatal("windows is not (yet) supported.")
	}

	if len(os.Args) > 1 {
		if os.Args[1] == bootstrapCommand {
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "file":
					runBootstrapFileCommand()
				case "dir":
					runBootstrapDirCommand()
				}
				return
			}
			printBootstrapCommandUsageErr()
			return
		}
	}

	if len(os.Args) > 2 {
		if os.Args[1] == "makefile" && os.Args[2] == "migrate" {
			migrateMakefile()
			return
		}
	}

	// check if zeus directory or Zeusfile exists
	stat, err := os.Stat(zeusDir)
	if err != nil {
		if stat, err = os.Stat(zeusfilePath); err != nil {
			if stat, err = os.Stat("Zeusfile"); err != nil {
				cLog.WithError(err).Error("no zeus directory or Zeusfile found.")
				cLog.Info("run 'zeus bootstrap dir' or 'zeus bootstrap file' to create a default one, or 'zeus makefile migrate' if you want to migrate from a GNU Makefile.")
				os.Exit(1)
			} else {
				// make sure its a file
				if stat.IsDir() {
					cLog.Fatal("Zeusfile is not a file")

				}
			}
		} else {
			// make sure its a file
			if stat.IsDir() {
				cLog.Fatal(zeusfilePath + " is not a file")
			}
		}
	} else {
		// make sure its a directory
		if !stat.IsDir() {
			cLog.Fatal("zeus/ is not a directory")
		}
	}

	// look for project data
	projectData, err = parseProjectData()
	if err != nil {
		cLog.WithError(err).Debug("error looking for project data")
		projectData = newData()
	}

	// look for project config
	conf, err = parseProjectConfig()
	if err != nil {

		cLog.WithError(err).Debug("failed to parse project config")

		// look for global config
		conf, err = parseGlobalConfig()
		if err != nil {
			cLog.WithError(err).Debug("failed to parse global config")
			cLog.Info("initializing default configuration")
			conf = newConfig()
			conf.update()
		}
	}

	// load persisted events from project data
	loadEvents()

	// validate aliases
	for name := range projectData.Aliases {
		err = validateAlias(name)
		if err != nil {
			Log.WithError(err).Fatal("failed to validate alias: ", name)
		}

		// add to completer
		completer.Children = append(completer.Children, readline.PcItem(name))
	}

	// get debug value from config
	debug = conf.Debug

	// handle debug mode for logger
	if debug {
		Log.Level = logrus.DebugLevel
	}

	if conf.DisableTimestamps {
		formatter := new(prefixed.TextFormatter)
		formatter.DisableTimestamp = true
		Log.Formatter = formatter
	}

	// disable colorized output if requested
	color.NoColor = !conf.Colors

	// disable ansi package colors manually
	if !conf.Colors {
		ansi.Red = ansi.ColorCode("off")
		ansi.Green = ansi.ColorCode("off")

		Log.Formatter = &prefixed.TextFormatter{
			DisableColors: true,
		}
	}

	if conf.LogToFile || conf.LogToFileColor {
		err := logToFile()
		if err != nil {
			cLog.WithError(err).Fatal("failed to set up logging to file")
		}
	}

	// init color profile
	switch conf.ColorProfile {
	case "dark":
		cp = darkProfile()
	case "light":
		cp = lightProfile()
	case "default":
		cp = defaultProfile()
	default:
		Log.Fatal(ErrUnknownColorProfile, " : ", conf.ColorProfile)
	}

	// set working directory
	workingDir, err = os.Getwd()
	if err != nil {
		cLog.WithError(err).Fatal("failed to get current directory name")
	}

	// only execute when using the interactive shell
	if len(os.Args) == 1 {

		clearScreen()

		// print ascii art
		asciiArt, err = assetBox.String("ascii_art.txt")
		if err != nil {
			cLog.WithError(err).Fatal("failed to get ascii art from rice box")
		}
		l.Println(cp.colorText + asciiArt + "\n")

		l.Println(cp.colorText + "Project Name: " + cp.colorPrompt + filepath.Base(workingDir) + cp.colorText + "\n")
		printAuthor()

		if projectData.BuildNumber > 0 {
			l.Println(cp.colorText + "BuildNumber: " + cp.colorPrompt + strconv.Itoa(projectData.BuildNumber) + cp.colorText + "\n")
		}

		// project infos
		printDeadline()
		listMilestones()
	}

	// start watchers when running in interactive mode
	if conf.Interactive {

		// watch config for changes
		go conf.watch("")

		if conf.AutoFormat {
			// watch zeus directory for changes
			go f.watchzeusDir("")
		}
	}

	// print makefile command overview
	if conf.MakefileOverview {
		printMakefileCommandOverview()
	}

	// check if a Zeusfile for the project exists
	err = parseZeusfile(zeusfilePath)
	if err == ErrFailedToReadZeusfile {

		// check if a Zeusfile for the project exists without the .yml extension
		err = parseZeusfile("Zeusfile")
		if err == ErrFailedToReadZeusfile {

			Log.Debug("no Zeusfile found. parsing zeusDir...")

			// create commandList from ZEUS dir
			findCommands()

			// watch scripts directory in interactive mode
			if conf.Interactive {
				go watchScripts("")
			}
		}
	}
	if err != nil && err != ErrFailedToReadZeusfile {
		Log.Error(err)
		println()
	}

	if conf.ProjectNamePrompt {
		// set shell prompt to project name
		zeusPrompt = filepath.Base(workingDir)
	}

	// handle commandline arguments
	handleArgs()

	// check if interactive mode is enabled in the config
	if conf.Interactive {

		if conf.WebInterface {
			go StartWebListener(true)
		}

		// handle OS Signals
		// all child processes need to be killed when theres an error
		handleSignals()

		// start interactive mode and start reading from stdin
		err = readlineLoop()
		if err != nil {
			cLog.WithError(err).Fatal("failed to read user input")
		}
	} else {
		if conf.PrintBuiltins {
			printBuiltins()
		}
		printCommands()
	}
}

// handle commandline arguments
func handleArgs() {

	var cLog = Log.WithField("prefix", "handleArgs")

	if len(os.Args) > 1 {

		var validCommand bool

		switch os.Args[1] {
		case helpCommand:
			if conf.PrintBuiltins {
				printBuiltins()
			}
			printCommands()

		case formatCommand:
			f.formatCommand()
		case "data":
			printProjectData()

		case aliasCommand:
			if len(os.Args) == 2 {
				printAliases()
				return
			}

			handleAliasCommand(os.Args[2:])

		case configCommand:
			handleConfigCommand(os.Args[2:])

		case versionCommand:
			l.Println(version)
		case updateCommand:
			updateZeus()
		case infoCommand:
			printProjectInfo()

		case colorsCommand:

			if len(os.Args) == 3 {
				handleColorsCommand(os.Args[1:])
			} else {
				printColorsUsageErr()
			}

		case authorCommand:
			handleAuthorCommand(os.Args[1:])

		case builtinsCommand:
			printBuiltins()

		case makefileCommand:
			handleMakefileCommand(os.Args[1:])
		case gitFilterCommand:
			handleGitFilterCommand(os.Args[1:])

		case createCommand:
			handleCreateCommand(os.Args[1:])
			os.Exit(0)

		default:
			handleSignals()

			commandMutex.Lock()

			// check if the command exists
			if cmd, ok := commands[os.Args[1]]; ok {
				commandMutex.Unlock()

				validCommand = true
				numCommands = getTotalCommandCount(cmd)

				err := cmd.Run(os.Args[2:])
				if err != nil {
					cLog.WithError(err).Error("failed to execute " + cmd.name)
					cleanup()
					os.Exit(1)
				}
			} else {
				commandMutex.Unlock()
			}

			// check if its a commandchain supplied with "" or ''
			if strings.Contains(os.Args[1], p.separator) {
				executeCommandChain(strings.Join(os.Args[1:], " "))
				return
			}

			// check if its an alias
			if command, ok := projectData.Aliases[os.Args[1]]; ok {
				handleLine(command)
				os.Exit(0)
			}

			if !validCommand {
				if !testingMode {
					cLog.Fatal("unknown command: ", os.Args[1])
				}
			}
		}
		if !testingMode {
			os.Exit(0)
		}
	}
}
