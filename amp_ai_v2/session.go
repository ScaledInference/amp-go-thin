package amp_ai_v2

import (
	"bytes"
	"context"
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
	amp   *Amp
	index int32
	SessionOpts
}

type SessionOpts struct {
	UserId, SessionId, AmpToken string
	Timeout                     time.Duration
	SessionLifetime             time.Duration
}

type CandidateField struct {
	Name   string
	Values []interface{}
}

type DecideResponse struct {
	Decision      map[string]interface{}
	AmpToken      string
	Fallback      bool // want this to be false (to indicate successful interaction with server)
	FailureReason string
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
	Ts              int64                  `json:"ts"`
	AmpToken        string                 `json:"ampToken,omitempty"`
	SessionLifetime int                    `json:"sessionLifetime"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	Decision        *decisionReq           `json:"decision,omitempty"`
}

type response struct {
	AmpToken string `json:"ampToken"`
	Decision string `json:"decision"`
}

func (s *Session) Observe(contextName string, contextProperties map[string]interface{}, timeOut time.Duration) (string, error) {
	if timeOut == 0 {
		timeOut = s.Timeout
	}
	ctx, cf := context.WithTimeout(context.Background(), timeOut)
	defer cf()

	if contextName == "" {
		return "", fmt.Errorf("context name can't be empty")
	}
	_, err := s.callAmpAgent(ctx, s.amp.getObserveUrl(s.UserId), &request{
		UserId:          s.UserId,
		SessionId:       s.SessionId,
		Name:            contextName,
		Index:           atomic.AddInt32(&s.index, 1),
		Ts:              time.Now().UnixNano() / 1e6,
		AmpToken:        s.AmpToken,
		SessionLifetime: int(s.SessionLifetime / time.Millisecond),
		Properties:      contextProperties,
	})
	return s.AmpToken, err
}

func (s *Session) DecideWithContext(contextName string, context map[string]interface{}, decisionName string, candidates []CandidateField, timeOut time.Duration) (*DecideResponse, error) {
	return s.callAmpAgentForDecide(s.amp.getDecideWithContextUrl(s.UserId), contextName, context, decisionName, candidates, timeOut)
}

func (s *Session) Decide(decisionName string, candidates []CandidateField, timeOut time.Duration) (*DecideResponse, error) {
	return s.callAmpAgentForDecide(s.amp.getDecideUrl(s.UserId), "", nil, decisionName, candidates, timeOut)
}

func (s *Session) callAmpAgentForDecide(endpoint, contextName string, contextProperties map[string]interface{},
	decisionName string, candidates []CandidateField, timeOut time.Duration) (*DecideResponse, error) {
	if timeOut == 0 {
		timeOut = s.Timeout
	}
	ctx, cf := context.WithTimeout(context.Background(), timeOut)
	defer cf()

	if contextName == "" {
		return nil, fmt.Errorf("context name can't be empty")
	}

	numCandidates := 1
	c := map[string]interface{}{}
	for _, f := range candidates {
		c[f.Name] = f.Values
		numCandidates *= len(f.Values)
	}
	if numCandidates > decideUpperLimit {
		return nil, fmt.Errorf("can't have more than %d candidates", decideUpperLimit)
	}

	req := &request{
		UserId:          s.UserId,
		SessionId:       s.SessionId,
		Name:            contextName,
		Index:           atomic.AddInt32(&s.index, 1),
		Ts:              time.Now().UnixNano() / 1e6,
		AmpToken:        s.AmpToken,
		SessionLifetime: int(s.SessionLifetime / time.Millisecond),
		Properties:      contextProperties,
		DecisionName:    decisionName,
		Decision: &decisionReq{
			Limit:      1,
			Candidates: []map[string]interface{}{c},
		},
	}

	r, err := s.callAmpAgent(ctx, endpoint, req)
	if err != nil {
		return &DecideResponse{
			Decision:      getCandidateByIndex(candidates, 0), // change this to the amp-agent decision if we ever get to that stage
			AmpToken:      s.AmpToken,                         // change this to the amp-agent amp token if we ever get to that stage
			Fallback:      true,
			FailureReason: err.Error(),
		}, nil
	}

	var decision map[string]interface{}
	err = json.Unmarshal([]byte(r.Decision), &decision)
	if err != nil {
		return &DecideResponse{
			Decision:      getCandidateByIndex(candidates, 0),
			AmpToken:      s.AmpToken,
			Fallback:      true,
			FailureReason: fmt.Sprintf("Can't unmarshal decision: %s", err),
		}, nil
	}

	return &DecideResponse{
		Decision: decision,
		AmpToken: s.AmpToken, // change this to the amp-agent amp token if we ever get to that stage
		Fallback: false,
	}, nil
}

func (s *Session) callAmpAgent(ctx context.Context, aaUrl string, req *request) (*response, error) {
	ba, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	pr, err := http.NewRequest(http.MethodPost, aaUrl, bytes.NewReader(ba))
	if err != nil {
		return nil, err
	}
	pr.Header.Set("Content-Type", "application/json")
	pr = pr.WithContext(ctx)
	resp, err := s.amp.httpClient.Do(pr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amp agent returned: %s : %s", resp.Status, string(body))
	}

	var r response
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response from server: %s", err)
	}

	if r.AmpToken == "" {
		log.Println("Received a response with no AmpToken")
	} else if s.amp.DontUseTokens {
		s.AmpToken = "CUSTOM"
	} else {
		s.AmpToken = r.AmpToken // Only the first call in the session changes the value of s.AmpToken
	}
	return &r, nil
}

func getCandidateByIndex(candidates []CandidateField, index int) map[string]interface{} {
	decision := map[string]interface{}{}
	partial := index
	for _, f := range candidates {
		decision[f.Name] = f.Values[partial%len(f.Values)]
		partial /= len(f.Values)
	}
	return decision
}
