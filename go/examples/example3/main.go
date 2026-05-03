package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/elk-language/go-prompt"
	pstrings "github.com/elk-language/go-prompt/strings"
	"github.com/hiveot/hivekit/go/examples/example3/wotcli"
	"github.com/hiveot/hivekit/go/utils"
)

func completer(d prompt.Document) ([]prompt.Suggest, pstrings.RuneNumber, pstrings.RuneNumber) {
	endIndex := d.CurrentRuneIndex()
	w := d.GetWordBeforeCursor()
	startIndex := endIndex - pstrings.RuneCount([]byte(w))

	s := []prompt.Suggest{
		// {Text: "h", Description: "Show help"},
		{Text: "d", Description: "Discover Things"},
		{Text: "l", Description: "List all known Things"},
		{Text: "r", Description: "Read Things <thingID>"},
		{Text: "q", Description: "Quit"},
	}
	return prompt.FilterHasPrefix(s, w, true), startIndex, endIndex
}

func main() {
	utils.SetLogging("warn", "")
	var p *prompt.Prompt

	// TODO:
	// 1. use factory with appenvironment and shared certs (with CA)
	// 2. use client cert auth from shared certs dir
	// 2b. or use pre-generated token

	// Ignore the certificate check just for this example. Dont do this at home.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	cli := wotcli.NewWotCLI()

	executor := func(input string) {
		if input == "" {
			return
		}
		fmt.Println("--- Command:", input, "---")

		parts := strings.SplitN(input, " ", 2)
		switch parts[0] {
		case "q":
			return
		case "d":
			cli.Discover()
			cli.ListThings()

			tds := cli.GetThings()
			for thingID := range tds {
				p.History().Add("r " + thingID)
			}

		case "l":
			cli.ListThings()
		case "r":
			if len(parts) > 1 {
				cli.ReadThing(parts[1])
			} else {
				fmt.Println("Missing ThingID")
			}
		default:
			fmt.Println("Unknown command " + input)
		}
		fmt.Println()
	}

	exitChecker := func(input string, breakline bool) bool {
		return breakline && input == "q"
	}

	fmt.Println("Type commands. Use Up/Down for history.")
	p = prompt.New(executor,
		prompt.WithPrefix("> "),
		prompt.WithCompleter(completer),
		prompt.WithExitChecker(exitChecker),
		// prompt.WithInitialText(cli.GetSummary()),
	)
	p.Run()
	println("Done\n")
}
