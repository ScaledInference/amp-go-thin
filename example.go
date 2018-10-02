package main

import (
	"fmt"
	"github.com/ScaledInference/amp-go-thin/amp_ai_v2"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		panic("Usage: example <projectKey> <ampAgent>")
	}
	ampOpts := amp_ai_v2.AmpOpts{
		ProjectKey:      os.Args[1], // e.g. "6f97ea165d886458"
		Domain:          os.Args[2], // e.g. "http://localhost:8100"
		SessionLifetime: 1800000,    // 30 minutes
		Timeout:         10 * time.Second,
	}
	amp, err := amp_ai_v2.NewAmp(ampOpts)
	if err != nil {
		panic(err)
	}
	// Create a session using the amp instance.
	sessionOpts := amp_ai_v2.SessionOpts{
		UserId: "XYZ",
	}
	firstSession, err := amp.CreateNewSession(sessionOpts)
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
	decisionAndToken, err := firstSession.DecideWithContext("AmpSession", context1, "Decide", candidates, 3*time.Second)
	if err != nil {
		panic(err)
	}
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
	// Observe the outcome with the default timeout
	clickProperties := map[string]interface{}{"url": "google.com", "pageNumber": 1}
	_, err = firstSession.Observe("Click", clickProperties, 0)
	if err != nil {
		fmt.Println("Observe call failed with an error: ", err)
	} else {
		fmt.Println("Observed the outcome successfully")
	}
}
