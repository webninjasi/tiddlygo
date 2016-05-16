package main

import (
	"io"
	"net/http"
	"os"
	"strings"
)

func isExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func parseOptions(optionsStr string) map[string]string {
	optsMap := make(map[string]string)
	opts := strings.Split(optionsStr, ";")

	for _, optStr := range opts {
		opt := strings.SplitN(optStr, "=", 2)

		if len(opt) == 2 {
			optsMap[opt[0]] = opt[1]
		}
	}

	return optsMap
}

// Source: http://stackoverflow.com/a/33853856
func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func toHttpAddr(addr string) string {
	addr_parts := strings.SplitN(addr, ":", 2)

	if addr_parts[0] == "" {
		addr_parts[0] = "127.0.0.1"
	}

	if addr_parts[1] == "" {
		return "http://" + addr_parts[0]
	}

	return "http://" + addr_parts[0] + ":" + addr_parts[1]
}
