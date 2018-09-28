package amp_ai_v2

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout         = 10000   // 10 seconds
	defaultSessionLifetime = 1800000 // 30 minutes
)

type Amp struct {
	timeOut, sessionLifetime                    int
	ssl                                         bool
	decideWithContextUrl, decideUrl, observeUrl string
	httpClient                                  *http.Client
}

func NewAmp(key, domain string, timeOut, sessionLifetime int) (*Amp, error) {
	if key == "" {
		return nil, fmt.Errorf("key can't be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain can't be empty")
	}
	if timeOut < 0 {
		return nil, fmt.Errorf("timeOut must be non-negative")
	}
	if timeOut == 0 {
		timeOut = defaultTimeout
	}
	if sessionLifetime < 0 {
		return nil, fmt.Errorf("sessionLifetime must be non-negative")
	}
	if sessionLifetime == 0 {
		sessionLifetime = defaultSessionLifetime
	}
	if !strings.HasPrefix(domain, "http") {
		return nil, fmt.Errorf(`domain "` + domain + `" must start with http or https`)
	}
	ssl := false
	if strings.HasPrefix(domain, "https") {
		ssl = true
	}

	return &Amp{timeOut: timeOut,
		sessionLifetime:      sessionLifetime,
		ssl:                  ssl,
		decideWithContextUrl: domain + "/api/core/v2/" + key + "/decideWithContextV2",
		decideUrl:            domain + "/api/core/v2/" + key + "/decideV2",
		observeUrl:           domain + "/api/core/v2/" + key + "/observeV2",
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10000,
				IdleConnTimeout: time.Minute,
			},
		},
	}, nil
}

func (a *Amp) CreateSession() (*Session, error) {
	return a.CreateNewSession("", "", 0, 0, "")
}

func (a *Amp) CreateNewSession(userId, sessionId string, timeOut, sessionLifetime int, ampToken string) (*Session, error) {
	if userId == "" {
		userId = generateRandomString()
	}
	if sessionId == "" {
		sessionId = generateRandomString()
	}
	if timeOut == 0 {
		timeOut = a.timeOut
	}
	if sessionLifetime == 0 {
		sessionLifetime = a.sessionLifetime
	}
	return &Session{
		amp:             a,
		userId:          userId,
		sessionId:       sessionId,
		timeOut:         timeOut,
		sessionLifetime: sessionLifetime,
		ampToken:        ampToken,
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
