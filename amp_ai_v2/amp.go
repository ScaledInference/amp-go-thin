package amp_ai_v2

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/serialx/hashring"
)

const (
	defaultTimeout         = 10 * time.Second
	defaultSessionLifetime = 30 * time.Minute
)

type AmpOpts struct {
	ProjectKey      string
	AmpAgents       []string
	Timeout         time.Duration
	SessionLifetime time.Duration
	DontUseTokens   bool
}

type Amp struct {
	timeOut    time.Duration
	httpClient *http.Client
	AmpOpts
	ring *hashring.HashRing
}

func NewAmp(opts AmpOpts) (*Amp, error) {
	if opts.ProjectKey == "" {
		return nil, fmt.Errorf("project key can't be empty")
	}
	if len(opts.AmpAgents) == 0 {
		return nil, fmt.Errorf("AmpAgents can't be empty")
	}
	if opts.Timeout < 0 {
		return nil, fmt.Errorf("timeOut must be non-negative")
	}
	if opts.Timeout == 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.SessionLifetime < 0 {
		return nil, fmt.Errorf("sessionLifetime must be non-negative")
	}
	if opts.SessionLifetime == 0 {
		opts.SessionLifetime = defaultSessionLifetime
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        5000,
			MaxIdleConnsPerHost: 5000,
			IdleConnTimeout:     time.Minute,
		},
		Timeout: opts.Timeout,
	}

	for _, aa := range opts.AmpAgents {
		if !strings.HasPrefix(aa, "http") {
			return nil, fmt.Errorf(`AmpAgent "` + aa + `" must start with http or https`)
		}
		req, err := http.NewRequest(http.MethodGet, aa+"/test/update_from_spa/"+opts.ProjectKey, nil)
		if err != nil {
			return nil, err
		}
		q := req.URL.Query()
		q.Add("session_life_time", strconv.FormatInt(int64(opts.SessionLifetime/time.Second), 10))
		req.URL.RawQuery = q.Encode()
		resp, err := httpClient.Do(req)
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
	}

	return &Amp{timeOut: opts.Timeout, httpClient: httpClient, AmpOpts: opts, ring: hashring.New(opts.AmpAgents)}, nil
}

func (a *Amp) CreateSession() (*Session, error) {
	return a.CreateNewSession(SessionOpts{})
}

func (a *Amp) CreateNewSession(opts SessionOpts) (*Session, error) {
	if opts.UserId == "" {
		opts.UserId = generateRandomString()
	}
	if opts.SessionId == "" {
		opts.SessionId = generateRandomString()
	}
	if opts.Timeout == 0 {
		opts.Timeout = a.Timeout
	}
	if opts.SessionLifetime == 0 {
		opts.SessionLifetime = a.SessionLifetime
	}
	if a.DontUseTokens {
		opts.AmpToken = "CUSTOM"
	}
	return &Session{
		amp:         a,
		SessionOpts: opts,
	}, nil
}

func (a *Amp) getDecideWithContextUrl(userId string) string {
	return a.selectAmpAgent(userId) + "/api/core/v2/" + a.ProjectKey + "/decideWithContextV2"
}

func (a *Amp) getDecideUrl(userId string) string {
	return a.selectAmpAgent(userId) + a.ProjectKey + "/decideV2"
}

func (a *Amp) getObserveUrl(userId string) string {
	return a.selectAmpAgent(userId) + a.ProjectKey + "/observeV2"
}

func (a *Amp) selectAmpAgent(userId string) (aa string) {
	aa, _ = a.ring.GetNode(userId)
	return aa
}

func generateRandomString() string {
	length := 16
	defaultCharset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	arr := make([]uint8, 16)
	for i := 0; i < length; i++ {
		arr[i] = defaultCharset[rand.Intn(len(defaultCharset))]
	}
	return string(arr)
}
