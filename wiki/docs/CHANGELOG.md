# ZEUS CHANGELOG

    ________ ____  __ __  ______  __  ___
    \___   // __ \|  |  \/  ___/  \\/^//^\
     /    /\  ___/|  |  /\___ \   //  \\ /
    /_____ \\___  >____//____  >  \\  /^\\
          \/    \/           \/   /\\/\ //\
                An Electrifying Build System

## 02.04.17

- switched to YAML for config file and project data
- implemented Zeusfile handling
- added an example Zeusfile
- updated README, changelog and tests
- sorting builtins alphabetically
- added test 50% badge
- updated LICENSE file
- implemented zeusfile bootstrapping
- implemented zeusfile to zeusDir migration

## 29.03.17

- added header watcher event to watch scripts and parse again on WRITE event
- updated tests
- working on parse error feedback

## 27.03.17

- handling strings for manipulating config
- command arguments accept name=val syntax
- added dateFormat for deadline and milestones to config
- refactored addEvent to keep eventID and formatterID when reloading events

## 22.03.17

- added zeus create to bootstrap single command

## 20.03.17

- warn about unknown config fields
- added filetypes for events

## 13.03.17

- improved UI
- improved events
- implemented dependencies & outputs

## 22.02.17

- updated godeps
- added wiki
- bootstrapped webinterface
- bootstrapped tests

## 21.02.17

- added sh/fileutil package
- disabled auto formatting by default to avoid issues with IDEs
- enabling DumpScriptOnError when enabling auto formatting
- ignoring WRITE events when there is a formatting job ongoing
- updated README

## 16.02.17

- fixed globals.sh generation when migrating makefiles
- fixed makefile target migration for targets that include blank lines for formatting reasons
- updated README
- fixed updating loglevel after changing to debug mode in the shell

## 14.02.17

- project structure cleanup
- created zeus_overview graffle
- added StopOnError and DumpScriptOnError config options
- updated README
- updated gif
- updated ascii