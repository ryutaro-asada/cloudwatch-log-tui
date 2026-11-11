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
	"github.com/aws/aws-sdk-go-v2/config"
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

func TestLoadOptions(t *testing.T) {
	tests := []struct {
		opts              Options
		profile, region   string
		wantOptionsLength int
	}{
		{Options{}, "", "", 0},
		{Options{Profile: "p"}, "p", "", 1},
		{Options{Region: "eu-west-1"}, "", "eu-west-1", 1},
		{Options{Profile: "p", Region: "eu-west-1"}, "p", "eu-west-1", 2},
	}
	for _, tt := range tests {
		fns := loadOptions(tt.opts)
		if len(fns) != tt.wantOptionsLength {
			t.Errorf("%+v: got %d options, want %d", tt.opts, len(fns), tt.wantOptionsLength)
		}
		var lo config.LoadOptions
		for _, fn := range fns {
			if err := fn(&lo); err != nil {
				t.Fatal(err)
			}
		}
		if lo.SharedConfigProfile != tt.profile || lo.Region != tt.region {
			t.Errorf("%+v: profile=%q region=%q, want %q %q", tt.opts, lo.SharedConfigProfile, lo.Region, tt.profile, tt.region)
		}
	}
}

func TestCredentialSource(t *testing.T) {
	envKeys := config.EnvConfig{Credentials: aws.Credentials{AccessKeyID: "a", SecretAccessKey: "s"}}
	tests := []struct {
		name    string
		profile string
		sources []any
		want    string
	}{
		{"flag wins over env keys", "p", []any{envKeys}, "p"},
		{"env keys", "", []any{envKeys, config.SharedConfig{Profile: "x"}}, "env"},
		{"web identity", "", []any{config.EnvConfig{WebIdentityTokenFilePath: "/t"}}, "env"},
		{"shared profile", "", []any{config.EnvConfig{SharedConfigProfile: "x"}, config.SharedConfig{Profile: "x"}}, "x"},
		{"AWS_PROFILE without config file", "", []any{config.EnvConfig{SharedConfigProfile: "x"}, config.SharedConfig{}}, "x"},
		{"nothing set", "", []any{config.EnvConfig{}, config.SharedConfig{}}, "default"},
	}
	for _, tt := range tests {
		if got := credentialSource(tt.profile, tt.sources); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func isolateAWSEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, k := range []string{"AWS_REGION", "AWS_DEFAULT_REGION", "AWS_PROFILE", "AWS_DEFAULT_PROFILE",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_WEB_IDENTITY_TOKEN_FILE"} {
		t.Setenv(k, "")
	}
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	return dir
}

func TestNewClientRegionAndProfile(t *testing.T) {
	dir := isolateAWSEnv(t)
	cfg := "[default]\n[profile p]\nregion = ap-northeast-1\n"
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := NewClient(ctx, Options{}); err == nil || !strings.Contains(err.Error(), "no AWS region") {
		t.Errorf("without a region: err = %v, want a missing region error", err)
	}

	c, err := NewClient(ctx, Options{Profile: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Profile() != "p" || c.Region() != "ap-northeast-1" {
		t.Errorf("--profile p: profile=%q region=%q", c.Profile(), c.Region())
	}

	t.Setenv("AWS_REGION", "us-west-2")
	c, err = NewClient(ctx, Options{Region: "eu-west-1"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Profile() != "default" || c.Region() != "eu-west-1" {
		t.Errorf("--region eu-west-1: profile=%q region=%q", c.Profile(), c.Region())
	}

	if _, err := NewClient(ctx, Options{Profile: "missing"}); err == nil {
		t.Error("unknown profile: want an error")
	}
}
