TiddlyGo
========

A simple web-server for TiddlyWiki written in Go.

Features
--------

* Viewing/storing TiddlyWiki files
* Creating a new TiddlyWiki
* Running commands before/after store request
* Committing changes on TiddlyWiki files (git)

Building
--------

#### Dependencies

* [Gorilla Mux](https://github.com/gorilla/mux) for routing
* [Trayhost](https://github.com/cratonica/trayhost) for the systray icon
* [2goarray](https://github.com/cratonica/2goarray) to convert embed icon file
* [rsrc](https://github.com/akavel/rsrc) to create rsrc.syso for windows binary icon

#### Windows

Generate icon.go and/or rsrc.syso (Change "386" to "amd64" to fix "incompatible with i386:x86-64" error):

	type tiddlygo.ico | %GOPATH%\bin\2goarray iconData main > icon.go
	%GOPATH%\bin\rsrc -ico tiddlygo.ico -arch 386

Build (with flags to hide console window):

	go build -ldflags -H=windowsgui

#### Linux

Generate icon.go:

	cat tiddlygo.ico | $GOPATH/bin/2goarray iconData main > icon.go

Build:

	go build

Config
------

Config file (tiddlygo.json) should be inside the working directory.

| Key         | Description                              | Default   |
|-------------|------------------------------------------|-----------|
| address     | Server address                           | :8080     |
| wikidir     | Path to store wiki files                 | wikidir   |
| templatedir | Path to find templates                   | templates |
| publicdir   | Path for static web files                | www       |
| username    | Username to use on store request         | tiddlygo  |
| password    | Password to use on store request         | tiddlygo  |
| events      | A js object to define actions for events |           |

Valid events:

* prestore
	* args: filename
* poststore
	* args: filename

Valid actions:

* cmd
* git
	* commit
	* add

You can use event args in action parameters (`$0` = first arg):

```json
[ "git", "add", "$0" ]
```

### Examples

Set username and password:

```json
{
	"username": "webninjasi",
	"password": "12345"
}
```

Commit changes on TiddlyWiki files (git):

```json
{
	"events":
	{
		"poststore":
		[
			[ "git", "add", "$0" ],
			[ "git", "commit" ]
		]
	}
}
```
