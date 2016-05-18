package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Address     string   `json:"address"`
	WikiDir     string   `json:"wikidir"`
	TemplateDir string   `json:"templatedir"`
	PublicDir   string   `json:"publicdir"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Events      EventMap `json:"events"`
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
		Address:     ":8080",
		WikiDir:     "wikidir",
		TemplateDir: "templates",
		PublicDir:   "www",
		Username:    "tiddlygo",
		Password:    "tiddlygo",
		Events:      EventMap{},
	}
}
