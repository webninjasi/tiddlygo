package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Address  string   `json:"address"`
	WikiDir  string   `json:"wikidir"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Events   EventMap `json:"events"`
}

func (cfg *Config) ReadFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}

	return nil
}

func NewConfig() *Config {
	return &Config{
		Address:  ":8080",
		WikiDir:  "wikidir",
		Username: "tiddlygo",
		Password: "tiddlygo",
		Events:   EventMap{},
	}
}
