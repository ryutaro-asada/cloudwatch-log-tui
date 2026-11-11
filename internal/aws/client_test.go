package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// fakeLogs serves canned CloudWatch Logs responses. pages maps the request
// nextToken ("" for the first call) to the JSON response body.
type fakeLogs struct {
	mu       sync.Mutex
	pages    map[string]string
	requests []map[string]any
}

func (f *fakeLogs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(body, &req)

	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	token, _ := req["nextToken"].(string)
	resp, ok := f.pages[token]
	if !ok {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"__type":"InvalidParameterException","message":"bad token"}`)
		return
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	fmt.Fprint(w, resp)
}

func newTestClient(t *testing.T, f *fakeLogs) *Client {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &Client{cwl: cwl.New(cwl.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(srv.URL),
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "a", SecretAccessKey: "s"}, nil
		}),
		RetryMaxAttempts: 1,
	})}
}

func eventsJSON(n int, prefix, nextToken string) string {
	var events []string
	for i := 0; i < n; i++ {
		events = append(events, fmt.Sprintf(`{"message":"%s%d","timestamp":%d}`, prefix, i, i))
	}
	s := `{"events":[` + strings.Join(events, ",") + `]`
	if nextToken != "" {
		s += `,"nextToken":"` + nextToken + `"`
	}
	return s + "}"
}

func eventInput() *LogEventInput {
	return &LogEventInput{
		Ctx:          context.Background(),
		LogGroupName: "g",
		StartTime:    time.Unix(0, 0),
		EndTime:      time.Unix(60, 0),
	}
}

func TestGetLogEventsFollowsPartialAndEmptyPages(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"":   eventsJSON(0, "a", "t1"),
		"t1": eventsJSON(3, "b", "t2"),
		"t2": eventsJSON(2, "c", ""),
	}}
	out, err := newTestClient(t, f).GetLogEvents(eventInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.LogEvents) != 5 {
		t.Fatalf("got %d events, want 5", len(out.LogEvents))
	}
	if out.NextToken != nil {
		t.Errorf("NextToken = %q, want nil at the end of the listing", *out.NextToken)
	}
}

func TestGetLogEventsStopsAtMaxLogEvents(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"":   eventsJSON(600, "a", "t1"),
		"t1": eventsJSON(400, "b", "t2"),
		"t2": eventsJSON(10, "c", ""),
	}}
	out, err := newTestClient(t, f).GetLogEvents(eventInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.LogEvents) != MaxLogEvents {
		t.Fatalf("got %d events, want %d", len(out.LogEvents), MaxLogEvents)
	}
	if aws.ToString(out.NextToken) != "t2" {
		t.Errorf("NextToken = %v, want t2", out.NextToken)
	}
	if got := f.requests[1]["limit"]; got != float64(400) {
		t.Errorf("second request limit = %v, want the remaining 400", got)
	}
}

func TestGetLogEventsStopsOnRepeatedToken(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"":   eventsJSON(1, "a", "t1"),
		"t1": eventsJSON(1, "b", "t1"),
	}}
	out, err := newTestClient(t, f).GetLogEvents(eventInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.LogEvents) != 2 || len(f.requests) != 2 {
		t.Fatalf("got %d events in %d requests, want 2 in 2", len(out.LogEvents), len(f.requests))
	}
}

func TestGetLogGroupsSkipsEmptyPages(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"":   `{"logGroups":[],"nextToken":"t1"}`,
		"t1": `{"logGroups":[{"logGroupName":"a"}],"nextToken":"t2"}`,
	}}
	out, err := newTestClient(t, f).GetLogGroups(&LogGroupInput{Ctx: context.Background(), FilterPattern: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.LogGroups) != 1 || aws.ToString(out.NextToken) != "t2" {
		t.Fatalf("got %d groups, token %v; want 1 group, token t2", len(out.LogGroups), out.NextToken)
	}
}

func TestGetLogGroupsRejectsInvalidPatternWithoutCallingAPI(t *testing.T) {
	f := &fakeLogs{}
	_, err := newTestClient(t, f).GetLogGroups(&LogGroupInput{Ctx: context.Background(), FilterPattern: "a b"})
	if err == nil {
		t.Fatal("want an error for a pattern with a space")
	}
	if len(f.requests) != 0 {
		t.Errorf("made %d requests, want 0", len(f.requests))
	}
}

func TestValidateLogGroupNamePattern(t *testing.T) {
	for _, p := range []string{"", "aws", "/aws/lambda/my-func_1.x#2"} {
		if err := ValidateLogGroupNamePattern(p); err != nil {
			t.Errorf("%q: unexpected error %v", p, err)
		}
	}
	for _, p := range []string{"a b", "a*", "日本", strings.Repeat("a", 513)} {
		if err := ValidateLogGroupNamePattern(p); err == nil {
			t.Errorf("%q: want an error", p)
		}
	}
}

func TestFormatMessage(t *testing.T) {
	for in, want := range map[string]string{
		"a":        "a\n",
		"a\n":      "a\n",
		"a\r\n":    "a\n",
		"a\nb\n\n": "a\nb\n",
		"":         "\n",
	} {
		if got := FormatMessage(cwlTypes.FilteredLogEvent{Message: aws.String(in)}); got != want {
			t.Errorf("FormatMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteLogEvents(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"":   `{"events":[{"message":"one"},{"message":"two\n"}],"nextToken":"t1"}`,
		"t1": `{"events":[{"message":"three\r\n"}]}`,
	}}
	dir := t.TempDir()
	input := eventInput()
	input.OutputFile = filepath.Join(dir, "out.txt")

	path, count, err := newTestClient(t, f).WriteLogEvents(input)
	if err != nil {
		t.Fatal(err)
	}
	if path != input.OutputFile || count != 3 {
		t.Errorf("got (%s, %d), want (%s, 3)", path, count, input.OutputFile)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "one\ntwo\nthree\n" {
		t.Errorf("file content = %q", got)
	}
	assertOnlyFiles(t, dir, "out.txt")
}

func TestWriteLogEventsKeepsExistingFileOnError(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{
		"": `{"events":[{"message":"new"}],"nextToken":"missing"}`,
	}}
	dir := t.TempDir()
	input := eventInput()
	input.OutputFile = filepath.Join(dir, "out.txt")
	if err := os.WriteFile(input.OutputFile, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := newTestClient(t, f).WriteLogEvents(input); err == nil {
		t.Fatal("want an error from the failing second page")
	}
	got, _ := os.ReadFile(input.OutputFile)
	if string(got) != "old\n" {
		t.Errorf("existing file was changed to %q", got)
	}
	assertOnlyFiles(t, dir, "out.txt")
}

func TestWriteLogEventsDefaultFileIsAbsolute(t *testing.T) {
	f := &fakeLogs{pages: map[string]string{"": `{"events":[]}`}}
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	path, count, err := newTestClient(t, f).WriteLogEvents(eventInput())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(path) || filepath.Base(path) != DefaultOutputFile || count != 0 {
		t.Errorf("got (%s, %d)", path, count)
	}
}

func assertOnlyFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != strings.Join(names, ",") {
		t.Errorf("files in %s = %v, want %v (temp file left behind?)", dir, got, names)
	}
}
