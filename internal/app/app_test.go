package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
	"github.com/ryutaro-asada/cloudwatch-log-tui/internal/state"
)

type call struct {
	op   string
	body map[string]any
}

// fakeLogs is a CloudWatch Logs endpoint. handler returns the JSON response
// for an operation, or "" for a 400 InvalidParameterException.
type fakeLogs struct {
	mu      sync.Mutex
	calls   []call
	handler func(r *http.Request, op string, body map[string]any) string
}

func (f *fakeLogs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "Logs_20140328.")

	f.mu.Lock()
	f.calls = append(f.calls, call{op, body})
	f.mu.Unlock()

	resp := f.handler(r, op, body)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	if resp == "" {
		w.WriteHeader(http.StatusBadRequest)
		resp = `{"__type":"InvalidParameterException","message":"invalid request"}`
	}
	fmt.Fprint(w, resp)
}

func (f *fakeLogs) callsOf(op string) []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []call
	for _, c := range f.calls {
		if c.op == op {
			out = append(out, c)
		}
	}
	return out
}

// defaultHandler serves two log groups with one stream each and one event.
func defaultHandler(_ *http.Request, op string, body map[string]any) string {
	switch op {
	case "DescribeLogGroups":
		return `{"logGroups":[{"logGroupName":"g1"},{"logGroupName":"g2"}]}`
	case "DescribeLogStreams":
		g := body["logGroupName"].(string)
		return `{"logStreams":[{"logStreamName":"` + g + `-s"}]}`
	case "FilterLogEvents":
		if p, _ := body["filterPattern"].(string); strings.HasPrefix(p, "{") && !strings.HasSuffix(p, "}") {
			return ""
		}
		return `{"events":[{"message":"hello"}]}`
	}
	return ""
}

type harness struct {
	t    *testing.T
	app  *App
	sim  tcell.SimulationScreen
	fake *fakeLogs
}

func newHarness(t *testing.T, handler func(*http.Request, string, map[string]any) string) *harness {
	t.Helper()
	fake := &fakeLogs{handler: handler}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client, err := awsr.NewClient(ctx, awsr.Options{})
	if err != nil {
		t.Fatal(err)
	}
	a := New(ctx, client)
	sim := tcell.NewSimulationScreen("UTF-8")
	a.tvApp.SetScreen(sim)

	done := make(chan error, 1)
	go func() { done <- a.Run() }()
	t.Cleanup(func() {
		a.Stop()
		if err := <-done; err != nil {
			t.Errorf("Run: %v", err)
		}
	})

	h := &harness{t: t, app: a, sim: sim, fake: fake}
	h.ui(func() { a.LoadLogGroups(state.Home) })
	h.waitFor("log groups", func() bool { return len(a.logGroups) > 0 })
	return h
}

// ui runs f on the event goroutine and waits for it.
func (h *harness) ui(f func()) {
	h.app.tvApp.QueueUpdate(f)
}

func (h *harness) waitFor(what string, cond func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var ok bool
		h.ui(func() { ok = cond() })
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.t.Fatalf("timed out waiting for %s", what)
}

func (h *harness) key(k tcell.Key) {
	h.sim.InjectKey(k, 0, tcell.ModNone)
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		h.sim.InjectKey(tcell.KeyRune, r, tcell.ModNone)
	}
}

func (h *harness) viewLog() string {
	var s string
	h.ui(func() { s = h.app.view.Widgets.LogEvent.ViewLog.GetText(true) })
	return s
}

func (h *harness) waitViewLog(substr string) {
	h.t.Helper()
	h.waitFor(fmt.Sprintf("view log containing %q", substr), func() bool {
		return strings.Contains(h.app.view.Widgets.LogEvent.ViewLog.GetText(true), substr)
	})
}

// waitCalls waits until op has been called n times.
func (h *harness) waitCalls(op string, n int) []call {
	h.t.Helper()
	h.waitFor(fmt.Sprintf("%d %s calls", n, op), func() bool { return len(h.fake.callsOf(op)) >= n })
	return h.fake.callsOf(op)
}

// openEvents selects log group row groupRow and log stream row streamRow
// (1 is "All Log Streams") and waits for the events page.
func (h *harness) openEvents(groupRow, streamRow int) {
	h.t.Helper()
	a := h.app
	h.ui(func() {
		a.tvApp.SetFocus(a.view.Widgets.LogGroup.Table)
		a.view.Widgets.LogGroup.Table.Select(groupRow, 0)
	})
	// Keys and h.ui calls travel on different channels, so wait for the
	// request each key triggers before relying on its effect.
	n := len(h.fake.callsOf("DescribeLogStreams"))
	h.key(tcell.KeyEnter)
	h.waitCalls("DescribeLogStreams", n+1)
	h.waitFor("log streams", func() bool { return len(a.logStreams) > 0 })
	h.ui(func() { a.view.Widgets.LogStream.Table.Select(streamRow, 0) })
	n = len(h.fake.callsOf("FilterLogEvents"))
	h.key(tcell.KeyEnter)
	h.waitCalls("FilterLogEvents", n+1)
	h.waitViewLog("hello")
}

func lastFilterCall(t *testing.T, h *harness) map[string]any {
	t.Helper()
	calls := h.fake.callsOf("FilterLogEvents")
	if len(calls) == 0 {
		t.Fatal("no FilterLogEvents call")
	}
	return calls[len(calls)-1].body
}

func msToLocal(v any) time.Time {
	return time.UnixMilli(int64(v.(float64))).Local()
}

func TestSearchWithInvalidCharacterShowsErrorWithoutRequest(t *testing.T) {
	h := newHarness(t, defaultHandler)
	before := len(h.fake.callsOf("DescribeLogGroups"))

	h.typeText("/a b")
	h.waitFor("error in status", func() bool {
		return strings.Contains(h.app.view.Widgets.Status.GetText(true), "Error")
	})
	for _, c := range h.fake.callsOf("DescribeLogGroups")[before:] {
		if p, _ := c.body["logGroupNamePattern"].(string); strings.Contains(p, " ") {
			t.Errorf("sent invalid pattern %q", p)
		}
	}
}

func TestStaleLogGroupResponseIsDropped(t *testing.T) {
	release := make(chan struct{})
	h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
		if op == "DescribeLogGroups" && body["logGroupNamePattern"] == "a" {
			select {
			case <-release:
			case <-r.Context().Done():
			}
			return `{"logGroups":[{"logGroupName":"stale"}]}`
		}
		if op == "DescribeLogGroups" && body["logGroupNamePattern"] == "ab" {
			return `{"logGroups":[{"logGroupName":"fresh"}]}`
		}
		return defaultHandler(r, op, body)
	})

	h.typeText("/a")
	h.waitCalls("DescribeLogGroups", 2)
	h.typeText("b")
	name := func() string {
		var s string
		h.ui(func() { s = h.app.view.Widgets.LogGroup.Table.GetCell(1, 0).Text })
		return s
	}
	h.waitFor("fresh result", func() bool { return h.app.view.Widgets.LogGroup.Table.GetCell(1, 0).Text == "fresh" })
	close(release)
	time.Sleep(200 * time.Millisecond)
	if got := name(); got != "fresh" {
		t.Errorf("table shows %q after the stale response arrived", got)
	}
}

func TestEnterOnPagingRowDoesNotSelectGroup(t *testing.T) {
	h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
		if op == "DescribeLogGroups" {
			if body["nextToken"] == "t2" {
				return `{"logGroups":[{"logGroupName":"g3"}]}`
			}
			return `{"logGroups":[{"logGroupName":"g1"}],"nextToken":"t2"}`
		}
		return defaultHandler(r, op, body)
	})
	h.ui(func() {
		ref, _ := rowRefAt(h.app.view.Widgets.LogGroup.Table, 2)
		if ref.kind != rowNext {
			t.Fatalf("row 2 = %+v, want the next page row", ref)
		}
	})
	h.typeText("j")
	h.waitFor("page 2", func() bool { return h.app.view.Widgets.LogGroup.Table.GetCell(2, 0).Text == "g3" })
	h.ui(func() { h.app.view.Widgets.LogGroup.Table.Select(1, 0) }) // PrevPage row
	h.waitFor("page 1", func() bool { return h.app.view.Widgets.LogGroup.Table.GetCell(1, 0).Text == "g1" })
	h.ui(func() { h.app.view.Widgets.LogGroup.Table.Select(2, 0) })
	h.key(tcell.KeyEnter)
	time.Sleep(200 * time.Millisecond)
	for _, c := range h.fake.callsOf("DescribeLogStreams") {
		if c.body["logGroupName"] != "g1" && c.body["logGroupName"] != "g3" {
			t.Errorf("listed streams of %v", c.body["logGroupName"])
		}
	}
}

func TestEmptyNextPageKeepsCurrentPage(t *testing.T) {
	h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
		if op == "DescribeLogGroups" {
			if body["nextToken"] == "t2" {
				return `{"logGroups":[]}`
			}
			return `{"logGroups":[{"logGroupName":"g1"}],"nextToken":"t2"}`
		}
		return defaultHandler(r, op, body)
	})
	h.typeText("j") // onto the NextPage row
	h.waitCalls("DescribeLogGroups", 2)
	h.waitFor("next page row removed", func() bool { return h.app.view.Widgets.LogGroup.Table.GetRowCount() == 2 })
	h.ui(func() {
		table := h.app.view.Widgets.LogGroup.Table
		if got := table.GetCell(1, 0).Text; got != "g1" {
			t.Errorf("row 1 = %q, want g1", got)
		}
		if h.app.state.LogGroup.HasPrev() {
			t.Error("page 1 has a previous page")
		}
	})
}

func TestStreamNamedLikeAllRowIsAnItem(t *testing.T) {
	h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
		if op == "DescribeLogStreams" {
			return `{"logStreams":[{"logStreamName":"All Log Streams"}]}`
		}
		return defaultHandler(r, op, body)
	})
	h.openEvents(1, 2)
	streams, _ := lastFilterCall(t, h)["logStreamNames"].([]any)
	if len(streams) != 1 || streams[0] != "All Log Streams" {
		t.Errorf("logStreamNames = %v", streams)
	}
}

func TestStreamSelectionIsResetForAnotherGroup(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 2)
	if s := lastFilterCall(t, h)["logStreamNames"]; fmt.Sprint(s) != "[g1-s]" {
		t.Fatalf("first query streams = %v", s)
	}
	h.key(tcell.KeyEscape)
	h.waitFor("log stream page", func() bool { return h.app.tvApp.GetFocus() == h.app.view.Widgets.LogStream.Table })
	h.ui(func() {
		if got := h.app.view.Widgets.LogStream.Table.GetCell(2, 0).Text; got != "x" {
			t.Errorf("selected mark = %q, want x", got)
		}
	})

	n := len(h.fake.callsOf("FilterLogEvents"))
	h.openEvents(2, 2)
	calls := h.waitCalls("FilterLogEvents", n+1)
	body := calls[len(calls)-1].body
	if body["logGroupName"] != "g2" || fmt.Sprint(body["logStreamNames"]) != "[g2-s]" {
		t.Errorf("second query = %v %v, want g2 [g2-s]", body["logGroupName"], body["logStreamNames"])
	}
}

func TestHourAndMinuteZero(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)

	pick := func(text string) {
		n := len(h.fake.callsOf("FilterLogEvents"))
		h.key(tcell.KeyEnter)
		h.typeText(text)
		h.key(tcell.KeyEnter)
		h.waitCalls("FilterLogEvents", n+1)
	}
	h.key(tcell.KeyTab) // month
	h.key(tcell.KeyTab) // day
	h.key(tcell.KeyTab) // hour
	pick("0")
	if got := msToLocal(lastFilterCall(t, h)["startTime"]).Hour(); got != 0 {
		t.Errorf("start hour = %d, want 0 (reload must use the new option)", got)
	}
	h.key(tcell.KeyTab) // minute
	pick("0")
	start := msToLocal(lastFilterCall(t, h)["startTime"])
	if start.Hour() != 0 || start.Minute() != 0 {
		t.Errorf("start = %v, want 00:00", start)
	}
	h.waitViewLog("hello")
}

func TestDayIsClampedWhenMonthChanges(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	a := h.app
	year := time.Now().Year()
	h.ui(func() {
		a.state.LogEvent.SetTime(state.Start, state.Month, 3)
		a.state.LogEvent.SetTime(state.Start, state.Day, 31)
		a.state.LogEvent.SetTime(state.End, state.Year, year)
		a.state.LogEvent.SetTime(state.End, state.Month, 12)
		a.state.LogEvent.SetTime(state.End, state.Day, 31)
		a.setDefaultDropDownLogEvents()
	})

	h.key(tcell.KeyTab) // month
	h.key(tcell.KeyEnter)
	h.typeText("2")
	h.key(tcell.KeyEnter)
	want := state.DaysIn(year, 2)
	h.waitFor("day clamp", func() bool {
		_, text := a.view.Widgets.LogEvent.StartDay.GetCurrentOption()
		return text == fmt.Sprint(want) && a.view.Widgets.LogEvent.StartDay.GetOptionCount() == want
	})
	h.waitFor("query with the clamped day", func() bool {
		calls := h.fake.callsOf("FilterLogEvents")
		start := msToLocal(calls[len(calls)-1].body["startTime"])
		return start.Month() == time.February && start.Day() == want
	})
}

func TestStartAfterEndShowsError(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	n := len(h.fake.callsOf("FilterLogEvents"))

	h.ui(func() {
		h.app.view.Widgets.LogEvent.StartYear.SetCurrentOption(h.app.view.Widgets.LogEvent.StartYear.GetOptionCount() - 1)
		h.app.tvApp.SetFocus(h.app.view.Widgets.LogEvent.EndYear)
	})
	h.typeText("k") // end year one year back
	h.waitViewLog("is after end time")
	if got := len(h.fake.callsOf("FilterLogEvents")); got > n+1 {
		t.Errorf("sent %d requests for an invalid range", got-n-1)
	}
}

func TestJKOnClosedDropDown(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	dd := h.app.view.Widgets.LogEvent.StartMonth
	h.key(tcell.KeyTab)
	var before int
	h.ui(func() { before, _ = dd.GetCurrentOption() })
	r, delta := 'j', 1
	if before == dd.GetOptionCount()-1 {
		r, delta = 'k', -1
	}
	h.typeText(string(r))
	h.waitFor("option change", func() bool { i, _ := dd.GetCurrentOption(); return i == before+delta })
	h.ui(func() {
		if dd.IsOpen() {
			t.Error("j/k opened the list")
		}
	})
}

func TestFilterIsAppliedOnEnter(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	h.ui(func() { h.app.tvApp.SetFocus(h.app.view.Widgets.LogEvent.FilterPattern) })
	n := len(h.fake.callsOf("FilterLogEvents"))

	h.typeText("{ $.")
	h.key(tcell.KeyEnter)
	h.waitViewLog("Error")
	calls := h.fake.callsOf("FilterLogEvents")
	if len(calls) != n+1 {
		t.Fatalf("got %d requests, want 1 on Enter only", len(calls)-n)
	}
	if p := calls[n].body["filterPattern"]; p != "{ $." {
		t.Errorf("filterPattern = %v", p)
	}

	n = len(calls)
	h.typeText(" }")
	h.key(tcell.KeyEnter)
	h.waitCalls("FilterLogEvents", n+1)
	h.waitViewLog("hello")
	if !strings.Contains(h.viewLog(), "FilterPattern: { $. }") {
		t.Errorf("view does not show the applied pattern:\n%s", h.viewLog())
	}
}

func TestSaveReportsPathAndRejectsConcurrentSave(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
		if op == "FilterLogEvents" && body["limit"] == float64(10000) {
			once.Do(func() {
				select {
				case <-release:
				case <-r.Context().Done():
				}
			})
			return `{"events":[{"message":"a"},{"message":"b\n"}]}`
		}
		return defaultHandler(r, op, body)
	})
	h.openEvents(1, 1)
	out := filepath.Join(t.TempDir(), "out.txt")
	h.ui(func() {
		h.app.view.Widgets.LogEvent.OutputFile.SetText(out)
		h.app.tvApp.SetFocus(h.app.view.Widgets.LogEvent.SaveEventLog)
	})
	h.key(tcell.KeyEnter)
	h.waitCalls("FilterLogEvents", 2)
	h.key(tcell.KeyEnter)
	h.waitFor("concurrent save error", func() bool {
		return strings.Contains(h.app.view.Widgets.Status.GetText(true), "already running")
	})
	if !strings.Contains(h.viewLog(), "hello") {
		t.Error("save wiped the log view")
	}
	close(release)
	h.waitFor("save result", func() bool {
		return strings.Contains(h.app.view.Widgets.Status.GetText(true), "Saved 2 log events to "+out)
	})
	if n := len(h.fake.callsOf("FilterLogEvents")); n != 2 {
		t.Errorf("FilterLogEvents calls = %d, want 2 (one load, one save)", n)
	}
}

func TestHeaderShowsAccount(t *testing.T) {
	for _, tt := range []struct {
		name, resp, want string
	}{
		{"success", `<GetCallerIdentityResponse><GetCallerIdentityResult><Account>123456789012</Account></GetCallerIdentityResult></GetCallerIdentityResponse>`, "Account: 123456789012"},
		{"failure", "", "Account: failed to get caller identity"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, func(r *http.Request, op string, body map[string]any) string {
				if op == "" {
					return tt.resp
				}
				return defaultHandler(r, op, body)
			})
			h.ui(h.app.LoadAccount)
			h.waitFor("account in header", func() bool {
				return strings.Contains(h.app.view.Widgets.Header.GetText(true), tt.want)
			})
			if got := h.app.view.Widgets.Header.GetText(true); !strings.Contains(got, "Profile: env  Region: us-east-1") {
				t.Errorf("header = %q", got)
			}
		})
	}
}

func TestOrderToggle(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	if got := lastFilterCall(t, h)["startFromHead"]; got != false {
		t.Errorf("default startFromHead = %v, want false (newest first)", got)
	}
	if !strings.Contains(h.viewLog(), "Order: newest first") {
		t.Errorf("view does not show the order:\n%s", h.viewLog())
	}

	n := len(h.fake.callsOf("FilterLogEvents"))
	h.typeText("o")
	h.waitCalls("FilterLogEvents", n+1)
	if got := lastFilterCall(t, h)["startFromHead"]; got != nil {
		t.Errorf("after toggle startFromHead = %v, want unset (oldest first)", got)
	}
	h.waitViewLog("Order: oldest first")

	h.ui(func() { h.app.tvApp.SetFocus(h.app.view.Widgets.LogEvent.FilterPattern) })
	h.typeText("o")
	h.waitFor("typed filter", func() bool { return h.app.view.Widgets.LogEvent.FilterPattern.GetText() == "o" })
	if h.app.state.LogEvent.Order() != state.OldestFirst {
		t.Error("typing o in the filter field toggled the order")
	}
}

func TestNewestFirstFallsBackBefore2024(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	n := len(h.fake.callsOf("FilterLogEvents"))
	h.ui(func() {
		h.app.state.LogEvent.SetTime(state.Start, state.Year, 2023)
		h.app.LoadLogEvents()
	})
	h.waitCalls("FilterLogEvents", n+1)
	if got := lastFilterCall(t, h)["startFromHead"]; got != nil {
		t.Errorf("startFromHead = %v, want unset for a start time before 2024", got)
	}
	h.waitViewLog("Order: oldest first (newest first needs a start time on or after 2024-01-01 00:00 UTC)")
	h.waitFor("fallback hint", func() bool {
		return strings.Contains(h.app.view.Widgets.LogEvent.Hints.GetText(true), "newest first needs")
	})
}

func TestTimePresets(t *testing.T) {
	h := newHarness(t, defaultHandler)
	h.openEvents(1, 1)
	w := h.app.view.Widgets.LogEvent
	for _, p := range timePresets {
		n := len(h.fake.callsOf("FilterLogEvents"))
		h.typeText(string(p.key))
		h.waitCalls("FilterLogEvents", n+1)
		if calls := h.fake.callsOf("FilterLogEvents"); len(calls) != n+1 {
			t.Fatalf("preset %s: %d FilterLogEvents calls, want exactly 1 more", p.label, len(calls)-n)
		}
		body := lastFilterCall(t, h)
		start, end := msToLocal(body["startTime"]), msToLocal(body["endTime"])
		if got := end.Sub(start).Round(time.Minute); got != p.d+time.Minute {
			t.Errorf("preset %s: end - start = %v, want %v plus the end minute", p.label, got, p.d)
		}
		if d := time.Since(end); d < -time.Minute || d > time.Minute {
			t.Errorf("preset %s: end = %v, want about now", p.label, end)
		}
		var startText string
		h.ui(func() {
			parts := make([]string, 0, 5)
			for _, dd := range []interface{ GetCurrentOption() (int, string) }{w.StartYear, w.StartMonth, w.StartDay, w.StartHour, w.StartMinute} {
				_, s := dd.GetCurrentOption()
				parts = append(parts, s)
			}
			startText = strings.Join(parts, " ")
		})
		if want := fmt.Sprintf("%d %d %d %d %d", start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute()); startText != want {
			t.Errorf("preset %s: start drop-downs = %q, want %q", p.label, startText, want)
		}
	}
}
