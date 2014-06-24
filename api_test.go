package linode

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const testAPIKey = "abc123"

func newTestClient() *Client {
	return NewClient(testAPIKey)
}

func TestNewRequest(t *testing.T) {
	c := newTestClient()
	r := c.NewRequest()
	if r.client != *c {
		t.Error("incorrect request client")
	}
}

func TestRequestURLsEmpty(t *testing.T) {
	c := newTestClient()
	r := c.NewRequest()
	urls, err := r.URLs()
	if err != nil {
		t.Error("unexpected error", err)
	}
	if len(urls) != 0 {
		t.Error("expected", 0, "given", len(urls))
	}
}

func testURL(requestArray string) string {
	return fmt.Sprintf("%s?api_action=batch&api_key=%s&api_requestArray=%s", apiEndpoint, testAPIKey, url.QueryEscape(requestArray))
}

// NOTE: maps, when converted to JSON, are sorted by their keys first. Take note when constructing expected urls
func TestRequestURLs(t *testing.T) {
	type testAction struct {
		method string
		params map[string]string
	}
	cases := []struct {
		actions  []testAction
		expected []string
		err      bool
	}{
		// test no params
		{
			[]testAction{testAction{"testAction", nil}},
			[]string{testURL(`[{"api_action":"testAction"}]`)},
			false,
		},
		// test multiple actions no params
		{
			[]testAction{
				testAction{"testAction1", nil},
				testAction{"testAction2", nil},
			},
			[]string{testURL(`[{"api_action":"testAction1"},{"api_action":"testAction2"}]`)},
			false,
		},
		// test with params
		{
			[]testAction{testAction{"testAction", map[string]string{"a": "bVal"}}},
			[]string{testURL(`[{"a":"bVal","api_action":"testAction"}]`)},
			false,
		},
		// test multiple actions with params
		{
			[]testAction{
				testAction{"testAction1", map[string]string{"x": "False"}},
				testAction{"testAction2", map[string]string{"y": "True"}},
			},
			[]string{testURL(`[{"api_action":"testAction1","x":"False"},{"api_action":"testAction2","y":"True"}]`)},
			false,
		},
	}

	for _, testCase := range cases {
		c := newTestClient()
		r := c.NewRequest()
		for _, a := range testCase.actions {
			r.AddAction(a.method, a.params)
		}
		urls, err := r.URLs()
		if testCase.err && err != nil {
			t.Error("unexpected error", err)
		}
		if len(urls) != len(testCase.expected) {
			t.Error("incorrect number of URLs", len(urls))
		}
		for i, u := range testCase.expected {
			if urls[i] != u {
				t.Error("expected", u, ", given", urls[i])
			}
		}
	}
}

func TestRequestURLsBatchLimit(t *testing.T) {
	iter := make([]interface{}, maxBatchRequests)

	c := newTestClient()
	r := c.NewRequest()
	for _ = range iter {
		r.AddAction("test", nil)
	}
	urls, err := r.URLs()
	if err != nil {
		t.Error("unexpected error", err)
	}
	if len(urls) != 1 {
		t.Error("expected", 1, "given", len(urls))
	}
	// add one more request which should cause 2nd url
	r.AddAction("straw", nil)
	urls, err = r.URLs()
	if err != nil {
		t.Error("unexpected error", err)
	}
	if len(urls) != 2 {
		t.Error("expected", 2, "given", len(urls))
	}
}

func newTestServer(status int, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		fmt.Fprintln(w, response)
	}))
}

func TestGetJSONWithJSONError(t *testing.T) {
	server := newTestServer(200, `[{"ERRORARRAY":[{"ERRORCODE":11,"ERRORMESSAGE":"RequestArray isn't valid JSON or WDDX"}],"DATA":{},"ACTION":"batch"}]`)
	var responses []response
	var errors []error

	responses, errors = getJSON(server.URL, responses, errors)
	if len(responses) != 0 {
		t.Error("expected", 0, "given", len(responses))
	}
	if len(errors) != 1 {
		t.Error("expected", 1, "given", len(errors))
	} else {
		expectedError := "[code: 11] RequestArray isn't valid JSON or WDDX"
		givenError := errors[0].Error()
		if givenError != expectedError {
			t.Error("expected", expectedError, "given", givenError)
		}
	}
}

func TestGetJSONWithJSONData(t *testing.T) {
	server := newTestServer(200, `[{"ERRORARRAY":[],"DATA":[{"ALERT_CPU_ENABLED":1,"ALERT_BWIN_ENABLED":1}],"ACTION":"linode.test"}]`)
	var responses []response
	var errors []error

	responses, errors = getJSON(server.URL, responses, errors)
	if len(errors) != 0 {
		t.Error("expected", 0, "given", len(errors))
		return
	}
	cases := []struct {
		action string
		data   string
	}{
		{"linode.test", `[{"ALERT_CPU_ENABLED":1,"ALERT_BWIN_ENABLED":1}]`},
	}
	if len(responses) != len(cases) {
		t.Error("expected", len(cases), "given", len(responses))
		return
	}

	for i, c := range cases {
		givenAction := responses[i].Action
		givenData := string(responses[i].Data)

		if c.action != givenAction {
			t.Error("expected", c.action, "given", givenAction)
		}

		if c.data != givenData {
			t.Error("expected", c.data, "given", givenData)
		}
	}
}

func TestGetJSONWithJSONMultipleData(t *testing.T) {
	server := newTestServer(200, `[{"ERRORARRAY":[],"DATA":{},"ACTION":"test.echo"},{"ERRORARRAY":[],"DATA":[{"LOCATION":"Dallas, TX, USA","DATACENTERID":2,"ABBR":"dallas"},{"LOCATION":"Fremont, CA, USA","DATACENTERID":3,"ABBR":"fremont"}],"ACTION":"avail.datacenters"}]`)
	var responses []response
	var errors []error

	responses, errors = getJSON(server.URL, responses, errors)
	if len(errors) != 0 {
		t.Error("expected", 0, "given", len(errors))
		return
	}

	cases := []struct {
		action string
		data   string
	}{
		{"test.echo", `{}`},
		{"avail.datacenters", `[{"LOCATION":"Dallas, TX, USA","DATACENTERID":2,"ABBR":"dallas"},{"LOCATION":"Fremont, CA, USA","DATACENTERID":3,"ABBR":"fremont"}]`},
	}

	if len(responses) != len(cases) {
		t.Error("expected", len(cases), "given", len(responses))
		return
	}

	for i, c := range cases {
		givenAction := responses[i].Action
		givenData := string(responses[i].Data)

		if c.action != givenAction {
			t.Error("expected", c.action, "given", givenAction)
		}

		if c.data != givenData {
			t.Error("expected", c.data, "given", givenData)
		}
	}
}

func TestGetJSONWithNon200(t *testing.T) {
	server := newTestServer(500, `[{"ERRORARRAY":[],"DATA":{},"ACTION":""}]`)
	var responses []response
	var errors []error

	responses, errors = getJSON(server.URL, responses, errors)
	if len(errors) != 1 {
		t.Error("expected", 1, "given", len(errors))
		return
	}
}

func TestGetJSONWithInvalidJSON(t *testing.T) {
	server := newTestServer(200, `i am no json`)
	var responses []response
	var errors []error

	responses, errors = getJSON(server.URL, responses, errors)
	if len(errors) != 1 {
		t.Error("expected", 1, "given", len(errors))
		return
	}
}
