package install

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// zoneService answers with the time zone of the address the request comes from, as plain text.
const zoneService = "https://ipapi.co/timezone/"

// DetectZone asks the internet which time zone this connection is in. It tells the service this machine's address, so
// it only runs when the person chooses it. The answer is checked against the zones the system knows.
func DetectZone(ctx context.Context, known []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zoneService, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "maxor-installer")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach the time zone service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the time zone service answered %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	return CheckZone(strings.TrimSpace(string(body)), known)
}

// CheckZone accepts an answer only if it is one of the zones the system knows.
func CheckZone(answer string, known []string) (string, error) {
	for _, z := range known {
		if z == answer {
			return z, nil
		}
	}
	return "", fmt.Errorf("the time zone service answered %q, which is not a zone this system knows", answer)
}
