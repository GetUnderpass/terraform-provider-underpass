// Package client talks to the Underpass API (/api/v1) with an API token.
// The endpoints are described at https://docs.getunderpass.com/manage/api.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultURL is Underpass's own coordination service.
const DefaultURL = "https://coord.getunderpass.com"

// Client calls one workspace's API: an API token only ever acts on the
// workspace it was created in.
type Client struct {
	base  string
	token string
	http  *http.Client
	// UserAgent names the provider and its version in each request.
	UserAgent string
}

// New makes a client for the service at baseURL with an API token
// ("tsapi-…").
func New(baseURL, token string) *Client {
	return &Client{
		base:  strings.TrimRight(baseURL, "/") + "/api/v1",
		token: token,
		http:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Error is an answer the API refused, with its message.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("Underpass API: %s (HTTP %d)", e.Message, e.Status)
}

// IsNotFound says the thing asked for doesn't exist (any more).
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// Do sends a request with body (JSON, or nothing when nil) and decodes the
// answer into out (when not nil).
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("can't reach Underpass at %s: %w", c.base, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		msg := strings.TrimSpace(string(data))
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			msg = e.Error
		}
		if msg == "" {
			msg = http.StatusText(res.StatusCode)
		}
		return &Error{Status: res.StatusCode, Message: msg}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// --- Auth keys ---

// AuthKey is an auth key as the API lists it (never its secret).
type AuthKey struct {
	ID          string    `json:"id"`
	Prefix      string    `json:"prefix"`
	Reusable    bool      `json:"reusable"`
	Ephemeral   bool      `json:"ephemeral"`
	Tags        []string  `json:"tags"`
	PreApproved bool      `json:"preApproved"`
	NoKeyExpiry bool      `json:"noKeyExpiry"`
	Valid       bool      `json:"valid"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// NewAuthKey is what an auth key is made with.
type NewAuthKey struct {
	Reusable      bool     `json:"reusable"`
	Ephemeral     bool     `json:"ephemeral"`
	PreApproved   bool     `json:"preApproved"`
	NoKeyExpiry   bool     `json:"noKeyExpiry"`
	Tags          []string `json:"tags"`
	ExpirySeconds int64    `json:"expirySeconds,omitempty"`
}

// CreateAuthKey makes a key and returns it with its secret, shown only now.
func (c *Client) CreateAuthKey(ctx context.Context, k NewAuthKey) (AuthKey, string, error) {
	var out struct {
		Key     string  `json:"key"`
		AuthKey AuthKey `json:"authKey"`
	}
	if k.Tags == nil {
		k.Tags = []string{}
	}
	err := c.Do(ctx, http.MethodPost, "/keys", k, &out)
	return out.AuthKey, out.Key, err
}

// AuthKey reads one key. Revoked keys (and ones cleaned up a month after
// they stopped working) are not found.
func (c *Client) AuthKey(ctx context.Context, id string) (AuthKey, error) {
	var out struct {
		Keys []AuthKey `json:"keys"`
	}
	if err := c.Do(ctx, http.MethodGet, "/keys", nil, &out); err != nil {
		return AuthKey{}, err
	}
	for _, k := range out.Keys {
		if k.ID == id {
			return k, nil
		}
	}
	return AuthKey{}, &Error{Status: http.StatusNotFound, Message: "no auth key " + id}
}

// RevokeAuthKey revokes a key; one already gone is fine.
func (c *Client) RevokeAuthKey(ctx context.Context, id string) error {
	err := c.Do(ctx, http.MethodDelete, "/keys/"+id, nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// --- Access controls ---

// Policy reads the access rules as JSON (rules, ssh, remote, clipboard,
// groups, tests), and whether the workspace has any of its own.
func (c *Client) Policy(ctx context.Context) (json.RawMessage, bool, error) {
	var out map[string]json.RawMessage
	if err := c.Do(ctx, http.MethodGet, "/policy", nil, &out); err != nil {
		return nil, false, err
	}
	var custom bool
	_ = json.Unmarshal(out["custom"], &custom)
	policy := map[string]json.RawMessage{}
	for _, k := range PolicyFields {
		if v, ok := out[k]; ok {
			policy[k] = v
		}
	}
	b, err := json.Marshal(policy)
	return b, custom, err
}

// PolicyFields are the parts of a policy that are written; the rest of the
// API's answer (test results, versions) is read only.
var PolicyFields = []string{"rules", "ssh", "remote", "clipboard", "groups", "tests"}

// SetPolicy replaces the access rules. The API refuses it when a test in
// the policy fails, with each failure in the message.
func (c *Client) SetPolicy(ctx context.Context, policy json.RawMessage) error {
	return c.Do(ctx, http.MethodPut, "/policy", policy, nil)
}

// ResetPolicy goes back to no rules: every machine reaches every other.
func (c *Client) ResetPolicy(ctx context.Context) error {
	return c.Do(ctx, http.MethodDelete, "/policy", nil, nil)
}

// --- Settings ---

// Settings are the workspace settings an API token may change. The others
// (remote control, clipboard, agents, terminal…) are people's, in the
// dashboard.
type Settings struct {
	KeyExpiryDays    *int      `json:"keyExpiryDays,omitempty"`
	KeyExpiryOff     *bool     `json:"keyExpiryOff,omitempty"`
	DeviceApproval   *bool     `json:"deviceApproval,omitempty"`
	NetworkLogs      *bool     `json:"networkLogs,omitempty"`
	AutoUpdate       *bool     `json:"autoUpdate,omitempty"`
	LanGames         *bool     `json:"lanGames,omitempty"`
	LanGamesMachines *[]string `json:"lanGamesMachines,omitempty"`
}

// SettingsNow are the settings as they are.
type SettingsNow struct {
	KeyExpiryDays    int      `json:"keyExpiryDays"`
	KeyExpiryOff     bool     `json:"keyExpiryOff"`
	DeviceApproval   bool     `json:"deviceApproval"`
	NetworkLogs      bool     `json:"networkLogs"`
	AutoUpdate       bool     `json:"autoUpdate"`
	LanGames         bool     `json:"lanGames"`
	LanGamesMachines []string `json:"lanGamesMachines"`
}

// Settings reads the workspace's settings.
func (c *Client) Settings(ctx context.Context) (SettingsNow, error) {
	var out SettingsNow
	err := c.Do(ctx, http.MethodGet, "/settings", nil, &out)
	return out, err
}

// UpdateSettings changes the settings given (nil fields stay as they are).
func (c *Client) UpdateSettings(ctx context.Context, s Settings) error {
	return c.Do(ctx, http.MethodPatch, "/settings", s, nil)
}

// --- DNS ---

// DNS is the workspace's DNS settings.
type DNS struct {
	Nameservers   []string            `json:"nameservers,omitempty"`
	SplitDNS      map[string][]string `json:"splitDns,omitempty"`
	SearchDomains []string            `json:"searchDomains,omitempty"`
	Name          string              `json:"name,omitempty"`
	HTTPS         bool                `json:"https,omitempty"`
	// Domain is the workspace's domain, from Name (read only).
	Domain string `json:"domain,omitempty"`
}

// DNS reads the DNS settings.
func (c *Client) DNS(ctx context.Context) (DNS, error) {
	var out DNS
	err := c.Do(ctx, http.MethodGet, "/dns", nil, &out)
	return out, err
}

// SetDNS replaces the DNS settings.
func (c *Client) SetDNS(ctx context.Context, d DNS) error {
	d.Domain = ""
	return c.Do(ctx, http.MethodPut, "/dns", d, nil)
}

// --- Machines ---

// Machine is a machine as the API shows it.
type Machine struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	FQDN              string    `json:"fqdn"`
	OS                string    `json:"os"`
	ClientVersion     string    `json:"clientVersion"`
	IPv4              string    `json:"ipv4"`
	User              string    `json:"user"`
	Tags              []string  `json:"tags"`
	Online            bool      `json:"online"`
	LastSeen          time.Time `json:"lastSeen"`
	KeyExpiryDisabled bool      `json:"keyExpiryDisabled"`
	Pending           bool      `json:"pending"`
	AdvertisedRoutes  []string  `json:"advertisedRoutes"`
	ApprovedRoutes    []string  `json:"approvedRoutes"`
}

// Machines lists the workspace's machines.
func (c *Client) Machines(ctx context.Context) ([]Machine, error) {
	var out struct {
		Nodes []Machine `json:"nodes"`
	}
	err := c.Do(ctx, http.MethodGet, "/nodes", nil, &out)
	return out.Nodes, err
}

// Machine reads one machine.
func (c *Client) Machine(ctx context.Context, id string) (Machine, error) {
	var out Machine
	err := c.Do(ctx, http.MethodGet, "/nodes/"+id, nil, &out)
	return out, err
}

// MachineChange is what can be changed on a machine (nil: unchanged).
type MachineChange struct {
	Name              *string `json:"name,omitempty"`
	KeyExpiryDisabled *bool   `json:"keyExpiryDisabled,omitempty"`
}

// UpdateMachine renames a machine or turns its key expiry on or off.
func (c *Client) UpdateMachine(ctx context.Context, id string, ch MachineChange) error {
	return c.Do(ctx, http.MethodPatch, "/nodes/"+id, ch, nil)
}

// ApproveMachine lets a machine waiting for device approval in.
func (c *Client) ApproveMachine(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodPost, "/nodes/"+id+"/approve", map[string]any{}, nil)
}

// SetApprovedRoutes sets which of the routes a machine offers are approved.
func (c *Client) SetApprovedRoutes(ctx context.Context, id string, routes []string) error {
	if routes == nil {
		routes = []string{}
	}
	return c.Do(ctx, http.MethodPost, "/nodes/"+id+"/routes", map[string]any{"approved": routes}, nil)
}
