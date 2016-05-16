package main

import (
	"errors"
	"log"
	"strings"
)

var (
	ErrInvalidEventData       = errors.New("Invalid event data!")
	ErrInvalidEventType       = errors.New("Invalid event type!")
	ErrInvalidEventActionType = errors.New("Invalid event action type!")
)

type EventMap map[string][][]string
type EventActionMap map[string][]EventActioner

type EventHandler struct {
	actions EventActionMap
}

func (this *EventHandler) Parse(events EventMap) {
	if this.actions == nil {
		this.actions = make(EventActionMap)
	}

	for raw_type, actions := range events {
		evt_type := strings.ToLower(raw_type)

		evt_actions, err := this.parseActions(evt_type, actions)
		if err != nil {
			log.Println("Warning: " + err.Error() + ": " + raw_type)
			continue
		}

		this.actions[evt_type] = evt_actions
	}
}

func (this EventHandler) parseActions(evt_type string, actions [][]string) ([]EventActioner, error) {
	if len(actions) == 0 {
		return nil, ErrInvalidEventData
	}

	if !validateEventType(evt_type) {
		return nil, ErrInvalidEventType
	}

	evt_actions := []EventActioner{}

	for _, data := range actions {
		action_type := strings.ToLower(data[0])

		evt_action, err := this.parseAction(action_type, data[1:])
		if err != nil {
			return nil, err
		}

		evt_actions = append(evt_actions, evt_action)
	}

	return evt_actions, nil
}

func (this EventHandler) parseAction(action_type string, data []string) (EventActioner, error) {
	evt_action := getEventAction(action_type, data)

	if evt_action == nil {
		return nil, ErrInvalidEventActionType
	}

	return evt_action, nil
}

func (this EventHandler) Handle(e_type string, args ...string) {
	actions, ok := this.actions[e_type]

	if !ok {
		return
	}

	var err error

	for _, action := range actions {
		err = action.Do(action.CombineArgs(args)...)
		if err != nil {
			log.Println("Warning: " + err.Error() + ": " + e_type)
			continue
		}
	}
}

func validateEventType(evt string) bool {
	evt = strings.ToLower(evt)

	switch evt {
	case "prestore", "poststore":
		return true
	}

	return false
}
