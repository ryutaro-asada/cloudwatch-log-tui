package state

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsr "github.com/ryutaro-asada/cloudwatch-log-tui/internal/aws"
)

func newLogEvent(start, end DateTime) *LogEvent {
	l := &LogEvent{}
	l.SetLogGroupSelected("g")
	l.times[Start] = start
	l.times[End] = end
	return l
}

func TestBeforeGetAcceptsZeroHourAndMinute(t *testing.T) {
	// Hour 0 and minute 0 used to be treated as unset (#17).
	l := newLogEvent(DateTime{2026, 1, 1, 0, 0}, DateTime{2026, 1, 1, 10, 0})
	var in awsr.LogEventInput
	if err := l.BeforeGet(&in); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local); !in.StartTime.Equal(want) {
		t.Errorf("StartTime = %v, want %v", in.StartTime, want)
	}
	if want := time.Date(2026, 1, 1, 10, 0, 59, 999e6, time.Local); !in.EndTime.Equal(want) {
		t.Errorf("EndTime = %v, want the end of the minute %v", in.EndTime, want)
	}
}

func TestBeforeGetValidation(t *testing.T) {
	ok := DateTime{2026, 3, 1, 12, 30}
	tests := []struct {
		name       string
		start, end DateTime
		group      string
		wantErr    string
	}{
		{"no group", ok, ok, "", "no log group"},
		{"zero year", DateTime{0, 3, 1, 0, 0}, ok, "g", "year"},
		{"month 13", DateTime{2026, 13, 1, 0, 0}, ok, "g", "month"},
		{"feb 30", DateTime{2026, 2, 30, 0, 0}, ok, "g", "date"},
		{"hour 24", ok, DateTime{2026, 3, 1, 24, 0}, "g", "hour"},
		{"minute 60", ok, DateTime{2026, 3, 1, 1, 60}, "g", "minute"},
		{"start after end", DateTime{2026, 3, 1, 12, 31}, ok, "g", "after"},
		{"same minute", ok, ok, "g", ""},
		{"leap day", DateTime{2024, 2, 29, 0, 0}, DateTime{2024, 2, 29, 1, 0}, "g", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LogEvent{logGroupName: tt.group, times: [2]DateTime{tt.start, tt.end}}
			err := l.BeforeGet(&awsr.LogEventInput{})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

// BeforeGet used to take the read lock twice. With a writer queued between
// the two RLock calls both goroutines blocked forever (#15, #18).
func TestBeforeGetDoesNotDeadlockWithWriters(t *testing.T) {
	l := newLogEvent(DateTime{2026, 1, 1, 0, 0}, DateTime{2026, 1, 1, 1, 0})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for i := 0; i < 200; i++ {
			wg.Add(2)
			go func() { defer wg.Done(); _ = l.BeforeGet(&awsr.LogEventInput{}) }()
			go func(i int) { defer wg.Done(); l.SetFilterPattern(fmt.Sprint(i)) }(i)
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock")
	}
}

func TestSetTimeClampsDay(t *testing.T) {
	l := newLogEvent(DateTime{2024, 1, 31, 0, 0}, DateTime{2024, 3, 31, 0, 0})
	l.SetTime(Start, Month, 2)
	if got := l.Time(Start).Day; got != 29 {
		t.Errorf("2024-02 day = %d, want 29", got)
	}
	l.SetTime(Start, Year, 2023)
	if got := l.Time(Start).Day; got != 28 {
		t.Errorf("2023-02 day = %d, want 28", got)
	}
	l.SetTime(End, Month, 4)
	if got := l.Time(End).Day; got != 30 {
		t.Errorf("04 day = %d, want 30", got)
	}
}

func TestDaysIn(t *testing.T) {
	for _, tt := range []struct{ y, m, want int }{
		{2024, 2, 29}, {2023, 2, 28}, {2100, 2, 28}, {2000, 2, 29}, {2026, 4, 30}, {2026, 12, 31},
	} {
		if got := DaysIn(tt.y, tt.m); got != tt.want {
			t.Errorf("DaysIn(%d, %d) = %d, want %d", tt.y, tt.m, got, tt.want)
		}
	}
}

func TestLogStreamSelection(t *testing.T) {
	l := &LogEvent{}
	l.SetLogGroupSelected("g1")
	for _, s := range []string{"a", "b", "a"} {
		if err := l.AddLogStream(s); err != nil {
			t.Fatal(err)
		}
	}
	got := l.GetLogStreamsSelected()
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("selected = %v, want [a b]", got)
	}

	got[0] = "changed"
	_ = append(got[:1], "aliased")
	if s := l.GetLogStreamsSelected(); strings.Join(s, ",") != "a,b" {
		t.Errorf("caller changes leaked into state: %v", s)
	}

	var in awsr.LogEventInput
	l.SetDefaultTime()
	if err := l.BeforeGet(&in); err != nil {
		t.Fatal(err)
	}
	in.LogStreamNames[0] = "changed"
	if s := l.GetLogStreamsSelected(); s[0] != "a" {
		t.Errorf("request shares the state slice: %v", s)
	}

	l.SetLogGroupSelected("g1")
	if len(l.GetLogStreamsSelected()) != 2 {
		t.Error("reselecting the same group cleared the streams")
	}
	l.SetLogGroupSelected("g2")
	if s := l.GetLogStreamsSelected(); len(s) != 0 {
		t.Errorf("streams of the previous group still selected: %v", s)
	}
}

func TestAddLogStreamLimit(t *testing.T) {
	l := &LogEvent{}
	for i := 0; i < awsr.MaxLogStreamNames; i++ {
		if err := l.AddLogStream(fmt.Sprint(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.AddLogStream("one-too-many"); err == nil {
		t.Error("want an error above the FilterLogEvents limit")
	}
	if err := l.AddLogStream("0"); err != nil {
		t.Errorf("re-adding a selected stream: %v", err)
	}
}

func TestPagination(t *testing.T) {
	var p Pagination
	if _, _, ok := p.Target(Next); ok {
		t.Fatal("Next before any page")
	}

	page, token, ok := p.Target(Home)
	if !ok || page != 1 || token != nil {
		t.Fatalf("Home = (%d, %v, %v)", page, token, ok)
	}
	// A short page with a token still has a next page.
	p.Apply(1, 3, aws.String("t2"))
	if !p.HasNext() || p.HasPrev() {
		t.Fatalf("page 1: next=%v prev=%v", p.HasNext(), p.HasPrev())
	}

	page, token, _ = p.Target(Next)
	if page != 2 || aws.ToString(token) != "t2" {
		t.Fatalf("Next = (%d, %v)", page, token)
	}
	p.Apply(2, 50, aws.String("t3"))
	if !p.HasPrev() || !p.HasNext() {
		t.Fatal("page 2 should have prev and next")
	}

	// An empty later page keeps page 2 current and ends the listing.
	page, _, _ = p.Target(Next)
	if p.Apply(page, 0, nil) {
		t.Fatal("empty page 3 applied")
	}
	if p.HasNext() {
		t.Error("still has next after an empty page")
	}
	page, token, ok = p.Target(Prev)
	if !ok || page != 1 || token != nil {
		t.Fatalf("Prev from page 2 = (%d, %v, %v)", page, token, ok)
	}

	p.Reset()
	if _, _, ok := p.Target(Next); ok {
		t.Error("Next after Reset")
	}
	if _, _, ok := p.Target(Prev); ok {
		t.Error("Prev after Reset")
	}
}

func TestLogGroupPatternChangeResetsPages(t *testing.T) {
	l := &LogGroup{}
	l.SetFilterPattern("a")
	page, _ := l.BeforeGet(&awsr.LogGroupInput{}, Home)
	l.AfterGet(page, &awsr.LogGroupOutput{NextToken: aws.String("t")})
	if !l.HasNext() {
		t.Fatal("want next page")
	}
	l.SetFilterPattern("ab")
	if _, ok := l.BeforeGet(&awsr.LogGroupInput{}, Next); ok {
		t.Error("Next uses a token from the previous pattern")
	}
}

func TestLogStreamGroupChangeResetsPages(t *testing.T) {
	l := &LogStream{}
	l.SetLogGroupSelected("g1")
	var in awsr.LogStreamInput
	page, _ := l.BeforeGet(&in, Home)
	l.AfterGet(page, &awsr.LogStreamOutput{NextToken: aws.String("t")})
	l.SetLogGroupSelected("g2")
	if _, ok := l.BeforeGet(&in, Next); ok {
		t.Error("Next uses a token from the previous group")
	}
	if _, ok := l.BeforeGet(&in, Home); !ok || in.LogGroupName != "g2" || in.NextToken != nil {
		t.Errorf("Home input = %+v", in)
	}
}

func TestBeforeGetOrder(t *testing.T) {
	end := DateTime{2026, 1, 1, 10, 0}
	tests := []struct {
		name   string
		start  time.Time
		toggle bool
		want   bool
	}{
		{"newest first by default", time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local), false, true},
		{"oldest first after toggle", time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local), true, false},
		{"falls back before 2024 UTC", time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC), false, false},
		{"2024-01-01 UTC is supported", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), false, true},
	}
	for _, tt := range tests {
		l := newLogEvent(dateTimeOf(tt.start.Local()), end)
		if tt.toggle {
			if got := l.ToggleOrder(); got != OldestFirst {
				t.Errorf("%s: ToggleOrder = %v, want OldestFirst", tt.name, got)
			}
		}
		var in awsr.LogEventInput
		if err := l.BeforeGet(&in); err != nil {
			t.Fatal(err)
		}
		if in.NewestFirst != tt.want {
			t.Errorf("%s: NewestFirst = %v, want %v", tt.name, in.NewestFirst, tt.want)
		}
	}
}

func TestSetLast(t *testing.T) {
	now := time.Date(2026, 1, 3, 0, 4, 30, 0, time.Local)
	tests := []struct {
		d     time.Duration
		start DateTime
	}{
		{5 * time.Minute, DateTime{2026, 1, 2, 23, 59}},
		{time.Hour, DateTime{2026, 1, 2, 23, 4}},
		{7 * 24 * time.Hour, DateTime{2025, 12, 27, 0, 4}},
	}
	for _, tt := range tests {
		var l LogEvent
		l.SetLast(tt.d, now)
		if got := l.Time(Start); got != tt.start {
			t.Errorf("SetLast(%v): start = %+v, want %+v", tt.d, got, tt.start)
		}
		if got, want := l.Time(End), (DateTime{2026, 1, 3, 0, 4}); got != want {
			t.Errorf("SetLast(%v): end = %+v, want %+v", tt.d, got, want)
		}
	}
}
