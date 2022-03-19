package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GargClient struct {
	email    string
	password string
	url      string
	token    string
	client   *http.Client
}

// make req is basic framework for making api calls
func (g *GargClient) makeReq(uri, method string, request io.Reader) (map[string]string, error) {
	req, err := http.NewRequest(method, g.url+uri, request)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Add("Accept", "application/json, text/plain, */*")

	if g.token != "" {
		req.Header.Add("Authorization", "Bearer "+g.token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 400 {
		return nil, fmt.Errorf("call to %s returned status %d \n", uri, resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respMap := make(map[string]string)
	err = json.Unmarshal(body, &respMap)
	return respMap, err
}

// NewGargClient will initialise a new connection with a token for backend
func NewGargClient(email, password, url string) (*GargClient, error) {
	if email == "" || password == "" || url == "" {
		return nil, fmt.Errorf("email / password / url fields are empty")
	}

	g := &GargClient{email: email,
		password: password,
		url:      url,
		client:   &http.Client{Timeout: 30 * time.Second},
	}

	err := g.Auth()

	return g, err
}

// Auth performs authentication and updates the token for use in subsequent calls
func (g *GargClient) Auth() error {

	creds := url.Values{}
	creds.Set("email", g.email)
	creds.Set("password", g.password)

	respMap, err := g.makeReq("/auth/authenticate", "POST", strings.NewReader(creds.Encode()))
	if err != nil {
		return err
	}

	if respMap["status"] != "200" {
		return fmt.Errorf("status message from garg is %s", respMap["status"])
	}

	g.token = respMap["message"]
	return nil

}

// ShowToken is just a helper function
func (g *GargClient) ShowToken() {
	fmt.Println(g.token)
}

// ListScenario finds the scenarios available
func (g *GargClient) ListScenarios() ([]byte, error) {
	resp, err := g.makeReq("/scenario/list", "GET", nil)

	if err != nil {
		return nil, err
	}

	contentByte, err := findKey(resp, "content")
	if err != nil {
		err = fmt.Errorf("%v in method ListScenario", err)
	}

	return contentByte, err
}

// StartScenario takes a scenario id and creates a new session and returns detail of session
func (g *GargClient) StartScenario(scenarioID string, accessCode string) ([]byte, error) {
	data := url.Values{}
	data.Set("scenario", scenarioID)
	data.Set("access_code", accessCode)
	resp, err := g.makeReq("/session/new", "POST", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	contentByte, err := findKey(resp, "content")
	if err != nil {
		err = fmt.Errorf("%v in method StartScenario", err)
	}

	return contentByte, err
}

// FindVMClaim will query the status of the VMClaim
func (g *GargClient) FindVMClaim(vmClaimID string) ([]byte, error) {
	resp, err := g.makeReq("/vmclaim/"+vmClaimID, "GET", nil)
	if err != nil {
		return nil, err
	}

	contentByte, err := findKey(resp, "content")
	if err != nil {
		err = fmt.Errorf("%v in method FindVMclaim", err)
	}

	return contentByte, err
}

func (g *GargClient) FinishSession(sessionID string) error {
	resp, err := g.makeReq("/session/"+sessionID+"/finished", "PUT", nil)
	if err != nil {
		return err
	}

	contentByte, err := findKey(resp, "message")
	if err != nil {
		return fmt.Errorf("%v in method FinishSession", err)
	}

	// in a successful finish the message is "updated session"
	if string(contentByte) != "updated session" {
		return fmt.Errorf("did not find expected message 'updated session' in FinishSession call")
	}

	return nil
}

// findKey is a helper function to just find the right key and perform base64 decode on the resp
func findKey(data map[string]string, key string) ([]byte, error) {
	b64Content, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("expect key %s not found in resp map", key)
	}

	contentByte, err := base64.StdEncoding.DecodeString(b64Content)
	return contentByte, err
}

func RegisterUser(email string, password string, accesscode string, address string) error {
	g := &GargClient{url: address,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	rego := url.Values{}
	rego.Set("email", email)
	rego.Set("password", password)
	rego.Set("access_code", accesscode)

	resp, err := g.makeReq("/auth/registerwithaccesscode", "POST", strings.NewReader(rego.Encode()))
	if err != nil {
		return err
	}

	if respMessage, ok := resp["message"]; !ok || respMessage != "created user" {
		return fmt.Errorf("error during user creation. either no response or no message doesnt match 'created user'")
	}
	return nil
}
