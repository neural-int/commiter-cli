package mlx

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSessionKeepsOneProcessAndEnforcesPhaseOrder(t *testing.T) {
	helper := shellHelper(t, `for profile in bounded-grouping bounded-category bounded-text; do
 IFS= read -r request || exit 1
 printf '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"%s","benchmark_metrics":{"loadSeconds":0}}\n' "$profile"
 done`)
	client := NewClient(helper)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	session, err := client.BeginSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, profile := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		r := validRequest()
		r.GenerationProfile = profile
		response, err := session.Generate(ctx, r)
		if err != nil || response.GeneratedJSON != "{}" {
			t.Fatalf("%s: %v", profile, err)
		}
	}
}
func TestSessionCancelsAndRejectsBadOrder(t *testing.T) {
	for _, badOrder := range []bool{false, true} {
		helper := shellHelper(t, `IFS= read -r request
 sleep 10`)
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		session, err := NewClient(helper).BeginSession(ctx)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		r := validRequest()
		r.GenerationProfile = "bounded-grouping"
		if badOrder {
			r.GenerationProfile = "bounded-text"
		}
		_, err = session.Generate(ctx, r)
		session.Close()
		cancel()
		if err == nil {
			t.Fatal("expected rejection")
		}
	}
}
func TestSessionRejectsMalformedFrame(t *testing.T) {
	helper := shellHelper(t, "IFS= read -r request\nprintf '%s\\n' '"+strings.Repeat("x", 100)+"'")
	s, e := NewClient(helper).BeginSession(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := validRequest()
	r.GenerationProfile = "bounded-grouping"
	if _, e = s.Generate(context.Background(), r); e == nil {
		t.Fatal("malformed accepted")
	}
}

func TestSessionRejectsUnterminatedResponse(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r request
 printf '%s' '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"bounded-grouping"}'`)
	s, e := NewClient(helper).BeginSession(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := validRequest()
	r.GenerationProfile = "bounded-grouping"
	if _, e = s.Generate(context.Background(), r); e == nil {
		t.Fatal("unterminated response accepted")
	}
}

func TestSessionRejectsCrashAfterFinalResponse(t *testing.T) {
	helper := shellHelper(t, `for profile in bounded-grouping bounded-category bounded-text; do
 IFS= read -r request || exit 1
 printf '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"%s"}\n' "$profile"
 done
 exit 7`)
	s, e := NewClient(helper).BeginSession(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for i, profile := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		r := validRequest()
		r.GenerationProfile = profile
		_, e = s.Generate(context.Background(), r)
		if (i == 2) != (e != nil) {
			t.Fatalf("phase %d: %v", i, e)
		}
	}
}
