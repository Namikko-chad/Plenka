package indexers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FlareSolverr (https://github.com/FlareSolverr/FlareSolverr) is an optional service the
// user runs next to the server. It opens a page in a headless browser, passes the
// Cloudflare check and returns the cookies and the browser's User-Agent; the session
// then repeats its request with them, as Jackett and Prowlarr do.

const flareSolverrTimeout = 60 * time.Second

var flareSolverrClient = &http.Client{Transport: sharedTransport}

type flareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Status  int `json:"status"`
		Cookies []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Domain string `json:"domain"`
		} `json:"cookies"`
		UserAgent string `json:"userAgent"`
	} `json:"solution"`
}

// solveChallenge asks FlareSolverr to open target (a GET, also for a POST request: the
// clearance cookie is per site) and takes over its cookies and User-Agent.
func (s *session) solveChallenge(ctx context.Context, solver, target string) error {
	raw, _ := json.Marshal(flareSolverrRequest{Cmd: "request.get", URL: target, MaxTimeout: int(flareSolverrTimeout / time.Millisecond)})
	ctx, cancel := context.WithTimeout(ctx, flareSolverrTimeout+10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(solver, "/")+"/v1", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("FlareSolverr: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := flareSolverrClient.Do(req)
	if err != nil {
		return fmt.Errorf("FlareSolverr is not reachable: %w", err)
	}
	defer resp.Body.Close()
	var out flareSolverrResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&out); err != nil {
		return fmt.Errorf("FlareSolverr: bad response (HTTP %d)", resp.StatusCode)
	}
	if out.Status != "ok" {
		msg := strings.TrimSpace(out.Message)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("FlareSolverr could not pass the check: %s", msg)
	}
	for _, c := range out.Solution.Cookies {
		if c.Name != "" && (c.Domain == "" || baseDomain(strings.TrimPrefix(c.Domain, ".")) == s.jar.site) {
			s.jar.set(c.Name, c.Value)
		}
	}
	if ua := strings.TrimSpace(out.Solution.UserAgent); ua != "" {
		// the clearance cookie is only valid together with the browser's User-Agent
		s.mu.Lock()
		s.ua = ua
		s.mu.Unlock()
	}
	return nil
}
