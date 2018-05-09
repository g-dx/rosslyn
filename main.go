package main

import (
	"io/ioutil"
	"os"
	"github.com/g-dx/rosslyn/slack"
	"log"
	"github.com/g-dx/rosslyn/ui"
	"github.com/nsf/termbox-go"
	"runtime/debug"
)

func main() {

	// Setup logging
	f, err := os.Create(os.ExpandEnv("${HOME}/.rosslyn/app.log"))
	if err != nil {
		panic(err)
	}
	logger := log.New(f, "", log.Ldate | log.Ltime)

	token, err := ioutil.ReadFile(os.ExpandEnv("${HOME}/.rosslyn/api-token"))
	if err != nil {
		panic(err)
	}

	//
	//
	// Configure the UI
	//
	//

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)
	termbox.SetOutputMode(termbox.Output256)

	//
	//
	// Connect to Slack
	//
	//
	defer func() {
		if err := recover(); err != nil {
			logger.Printf("Panic: %v\nStacktrace: %s\n", err, debug.Stack())
			panic(err)
		}
	}()
	ctrl := ui.NewController(logger, slack.NewApis(token))
	ctrl.Run()
}