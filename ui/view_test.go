package ui

import (
	"testing"
	"time"
	"reflect"
)

func TestTypingMonitorAddAndRemove(t *testing.T) {
	ch := make(chan func())
	d := time.Millisecond * 50
	ut := &UserTypingTimer{d, make(map[string]*time.Timer), ch }

	if len(ut.UsersTyping()) != 0 {
		t.Errorf("Got '%v', Wanted: 0", len(ut.UsersTyping()))
	}

	ut.Add("<user1>", func() { ut.Remove("<user1>") })
	got := ut.UsersTyping()
	want := []string{"<user1>"}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}

	time.Sleep(d) // Wait for <user1> to expire

	ut.Add("<user2>", func() { ut.Remove("<user2>") })
	got = ut.UsersTyping()
	want = append(want, "<user2>")
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}

	// Wait for timeout
	select {
	case f := <- ch:
		f()
	case <- time.After(100 * time.Millisecond):
		t.Fatal("Wanted: <timeout after 50ms>, Got: <none after 100ms>")
	}

	got = ut.UsersTyping()
	want = want[1:]
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}

	// Wait for timeout
	select {
	case f := <- ch:
		f()
	case <- time.After(100 * time.Millisecond):
		t.Fatal("Wanted: <timeout after 50ms>, Got: <none after 100ms>")
	}

	got = ut.UsersTyping()
	want = nil
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}
}

func TestTypingMonitorClear(t *testing.T) {
	ch := make(chan func())
	d := time.Millisecond * 50
	ut := &UserTypingTimer{d, make(map[string]*time.Timer), ch }

	ut.Add("<user1>", func() { ut.Remove("<user1>") })
	ut.Add("<user2>", func() { ut.Remove("<user2>") })

	got := ut.UsersTyping()
	want := []string{"<user1>", "<user2>"}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}

	ut.Clear()
	got = ut.UsersTyping()
	want = nil
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got '%v', Wanted: '%v'", got, want)
	}

}