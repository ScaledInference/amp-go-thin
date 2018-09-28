package main

import (
	"fmt"
	"github.com/ScaledInference/amp-go-thin/amp_ai_v2"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		panic("Usage: example <projectKey> <ampAgent>")
	}
	projectKey := os.Args[1]     // e.g. "6f97ea165d886458"
	ampAgent := os.Args[2]       // e.g. "http://localhost:8100"
	timeOutMilliseconds := 10000 // 10000 == 10 seconds
	sessionLifetime := 1800000   // 30 minutes
	amp, err := amp_ai_v2.NewAmp(projectKey, ampAgent, timeOutMilliseconds, sessionLifetime)
	if err != nil {
		panic(err)
	}
	// Create a session using the amp instance.
	firstSession, err := amp.CreateSession()
	if err != nil {
		panic(err)
	}
	// Prepare a context for making a decideWithContext call.
	context1 := map[string]interface{}{"browser_height": 1740, "browser_width": 360}
	// Prepare candidates for making a decideWithContext call.
	candidates := []amp_ai_v2.CandidateField{
		{"color", []interface{}{"red", "green", "blue"}},
		{"count", []interface{}{10, 100}},
	}
	// Make the decideWithContext api call.
	decisionAndToken, err := firstSession.DecideWithContext("AmpSession", context1, "Decide", candidates, 3000)
	// Look at the return value.
	a := decisionAndToken.AmpToken
	fmt.Printf("Returned ampToken: %s of length %d\n", a, len(a))
	fmt.Println("Returned decision:", decisionAndToken.Decision)
	if decisionAndToken.Fallback {
		fmt.Println("Decision NOT successfully obtained from amp-agent. Using a fallback instead.")
		fmt.Println("The reason is:", err)
	} else {
		fmt.Println("Decision successfully obtained from amp-agent")
	}
}
