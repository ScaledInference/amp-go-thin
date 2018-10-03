package amp_ai_v2

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout         = 10 * time.Second
	defaultSessionLifetime = 30 * time.Minute
)

type AmpOpts struct {
	ProjectKey, Domain string
	Timeout            time.Duration
	SessionLifetime    time.Duration
}

type Amp struct {
	timeOut                                     time.Duration
	ssl                                         bool
	decideWithContextUrl, decideUrl, observeUrl string
	httpClient                                  *http.Client
	AmpOpts
}

func NewAmp(opts AmpOpts) (*Amp, error) {
	if opts.ProjectKey == "" {
		return nil, fmt.Errorf("project key can't be empty")
	}
	if opts.Domain == "" {
		return nil, fmt.Errorf("domain can't be empty")
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
	if !strings.HasPrefix(opts.Domain, "http") {
		return nil, fmt.Errorf(`domain "` + opts.Domain + `" must start with http or https`)
	}
	ssl := false
	if strings.HasPrefix(opts.Domain, "https") {
		ssl = true
	}

	return &Amp{
		AmpOpts:              opts,
		ssl:                  ssl,
		decideWithContextUrl: opts.Domain + "/api/core/v2/" + opts.ProjectKey + "/decideWithContextV2",
		decideUrl:            opts.Domain + "/api/core/v2/" + opts.ProjectKey + "/decideV2",
		observeUrl:           opts.Domain + "/api/core/v2/" + opts.ProjectKey + "/observeV2",
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        5000,
				MaxIdleConnsPerHost: 5000,
				IdleConnTimeout:     time.Minute,
			},
			Timeout: opts.Timeout,
		},
	}, nil
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
	return &Session{
		amp:         a,
		SessionOpts: opts,
	}, nil
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
