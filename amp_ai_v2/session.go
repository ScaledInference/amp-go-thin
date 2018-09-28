package amp_ai_v2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	decideUpperLimit = 50
)

type Session struct {
	amp                         *Amp
	userId, sessionId, ampToken string
	index                       int32
	timeOut, sessionLifetime    int
}

type CandidateField struct {
	Name   string
	values []interface{}
}

type DecideResponse struct {
	Decision map[string]interface{}
	AmpToken string
	Fallback bool // want this to be false (to indicate successful interaction with server)
}

type decisionReq struct {
	Limit      int                      `json:"limit"`
	Candidates []map[string]interface{} `json:"candidates"`
}

type request struct {
	UserId          string                 `json:"userId"`
	SessionId       string                 `json:"sessionId"`
	DecisionName    string                 `json:"decisionName,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Index           int32                  `json:"index"`
	Ts              time.Time              `json:"ts"`
	AmpToken        string                 `json:"ampToken"`
	SessionLifetime int                    `json:"sessionLifetime"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	Decision        *decisionReq           `json:"Decision,omitempty"`
}

type response struct {
	AmpToken      string `json:"ampToken"`
	Decision      string `json:"decision"`
	Fallback      bool   `json:"fallback"`
	FailureReason string `json:"failureReason"`
}

func (s *Session) observe(contextName string, context map[string]interface{}, timeOut int) (string, error) {
	if timeOut == 0 {
		timeOut = s.timeOut
	}
	if contextName == "" {
		return "", fmt.Errorf("context name can't be empty")
	}
	_, err := s.callAmpAgent(s.amp.observeUrl, &request{
		UserId:          s.userId,
		SessionId:       s.sessionId,
		Name:            contextName,
		Index:           atomic.AddInt32(&s.index, 1),
		Ts:              time.Now(),
		AmpToken:        s.ampToken,
		SessionLifetime: s.sessionLifetime,
		Properties:      context,
	})
	return s.ampToken, err
}

func (s *Session) DecideWithContext(contextName string, context map[string]interface{}, decisionName string, candidates []CandidateField, timeOut int) (*DecideResponse, error) {
	return s.callAmpAgentForDecide(s.amp.decideWithContextUrl, contextName, context, decisionName, candidates, timeOut)
}

func (s *Session) decide(decisionName string, candidates []CandidateField, timeOut int) (*DecideResponse, error) {
	return s.callAmpAgentForDecide(s.amp.decideUrl, decisionName, nil, "", candidates, timeOut)
}

func (s *Session) callAmpAgentForDecide(
	endpoint, contextName string,
	context map[string]interface{},
	decisionName string,
	candidates []CandidateField,
	timeOut int) (*DecideResponse, error) {
	if timeOut == 0 {
		timeOut = s.timeOut
	}
	if contextName == "" {
		return nil, fmt.Errorf("context name can't be empty")
	}

	numCandidates := 1
	c := map[string]interface{}{}
	for _, f := range candidates {
		c[f.Name] = f.values
		numCandidates *= len(f.values)
	}
	if numCandidates > decideUpperLimit {
		return nil, fmt.Errorf("can't have more than %d candidates", decideUpperLimit)
	}

	req := &request{
		UserId:          s.userId,
		SessionId:       s.sessionId,
		Name:            contextName,
		Index:           atomic.AddInt32(&s.index, 1),
		Ts:              time.Now(),
		AmpToken:        s.ampToken,
		SessionLifetime: s.sessionLifetime,
		Properties:      context,
		DecisionName:    decisionName,
		Decision: &decisionReq{
			Limit:      1,
			Candidates: []map[string]interface{}{c},
		},
	}

	r, err := s.callAmpAgent(endpoint, req)
	if err != nil {
		return nil, err
	}

	if r.Fallback {
		return &DecideResponse{
			Decision: getCandidateByIndex(candidates, 0), // change this to the amp-agent decision if we ever get to that stage
			AmpToken: s.ampToken,                         // change this to the amp-agent amp token if we ever get to that stage
			Fallback: true,
		}, fmt.Errorf(r.FailureReason)
	}

	var decision map[string]interface{}
	json.Unmarshal([]byte(r.Decision), &decision)

	return &DecideResponse{
		Decision: decision,
		AmpToken: s.ampToken, // change this to the amp-agent amp token if we ever get to that stage
		Fallback: false,
	}, nil
}

func (s *Session) callAmpAgent(url string, req *request) (*response, error) {
	ba, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := s.amp.httpClient.Post(url, "application/json", bytes.NewReader(ba))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from server %d with reason: %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r response
	err = json.Unmarshal(body, r)
	if err != nil {
		return nil, err
	}

	if r.AmpToken == "" {
		log.Println("Received a response with no ampToken")
	} else {
		s.ampToken = r.AmpToken // Only the first call in the session changes the value of s.ampToken
	}
	return &r, err
}

func getCandidateByIndex(candidates []CandidateField, index int) map[string]interface{} {
	decision := map[string]interface{}{}
	partial := index
	for _, f := range candidates {
		decision[f.Name] = f.values[partial%len(f.values)]
		partial /= len(f.values)
	}
	return decision
}
