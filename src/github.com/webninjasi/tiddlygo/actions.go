package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrNoCommand = errors.New("No command specified to run!")
)

type EventActioner interface {
	Do(...string) error
	CombineArgs([]string) []string
}

type EventAction struct {
	data []string
}

func (this EventAction) Do(args ...string) error {
	return nil
}

func (this EventAction) CombineArgs(args []string) []string {
	combined := make([]string, len(this.data))
	copy(combined, this.data)

	for idx, dat := range this.data {
		if len(dat) >= 2 && strings.Index(dat, "$") == 0 {
			num, err := strconv.Atoi(dat[1:])
			if err == nil && len(args) > num && num >= 0 {
				combined[idx] = args[num]
			}
		}
	}

	return combined
}

type EventActionCmd struct {
	EventAction
}

func (this EventActionCmd) Do(args ...string) error {
	if len(args) == 0 {
		return ErrNoCommand
	}

	var err error

	if len(args) > 1 {
		_, err = runWikiCmd(exec.Command(args[0], args[1:]...))
	} else {
		_, err = runWikiCmd(exec.Command(args[0]))
	}

	return err
}

type EventActionGit struct {
	EventAction
}

func (this EventActionGit) Do(args ...string) (err error) {
	switch args[0] {
	case "add":
		filename := args[1]
		_, err = gitAdd(filename)
	case "commit":
		_, err = gitCommit("Update from tiddlygo", "")
	}

	return
}

func gitAdd(filename string) (string, error) {
	return runGitCmd(exec.Command("git", "add", filename))
}

func gitCommit(msg string, author string) (string, error) {
	if author != "" {
		return runGitCmd(exec.Command("git", "commit", "-m", msg, fmt.Sprintf("--author='%s <system@tiddlygo>'", author)))
	}

	return runGitCmd(exec.Command("git", "commit", "-m", msg))
}

func runCmd(cmd *exec.Cmd) (string, string, error) {
	var err error

	cmd.Dir, err = filepath.Abs(cfg.WikiDir)
	if err != nil {
		return "", "", err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	return stdout.String(), stderr.String(), err
}

func runWikiCmd(cmd *exec.Cmd) (string, error) {
	out, _, err := runCmd(cmd)

	return out, err
}

func runGitCmd(cmd *exec.Cmd) (string, error) {
	out, errstr, err := runCmd(cmd)

	if err != nil {
		if len(errstr) == 0 {
			err = nil
		} else {
			err = errors.New(errstr)
		}
	}

	return out, err
}

func getEventAction(action string, data []string) EventActioner {
	action = strings.ToLower(action)

	switch action {
	case "cmd":
		return EventActionCmd{EventAction{data}}
	case "git":
		return EventActionGit{EventAction{data}}
	}

	return nil
}
