package web

import (
	"testing"
	"time"
)

func TestSSEBroker_PublishBoardList(t *testing.T) {
	b := NewSSEBroker()
	ch, cancel := b.SubscribeBoardList()
	defer cancel()
	b.PublishBoardList()
	select {
	case ev := <-ch:
		if ev.Type != "board.list.updated" {
			t.Errorf("type = %q, want board.list.updated", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("no event received")
	}
}

func TestSSEBroker_SubscribeAllBoards(t *testing.T) {
	b := NewSSEBroker()
	ch, cancel := b.SubscribeAllBoards()
	defer cancel()
	b.Publish("welcome")
	select {
	case ev := <-ch:
		if ev.Type != "board.updated" {
			t.Errorf("type = %q, want board.updated", ev.Type)
		}
		if ev.Payload != "welcome" {
			t.Errorf("payload = %q, want welcome", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no event received")
	}
}

func TestSSEBroker_CancelStopsDelivery(t *testing.T) {
	b := NewSSEBroker()
	ch, cancel := b.SubscribeBoardList()
	cancel()
	b.PublishBoardList()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel closed or no delivery after cancel")
		}
	case <-time.After(50 * time.Millisecond):
		// Acceptable: nothing arrived.
	}
}

func TestSSEBroker_PublishStillNotifiesPerSlug(t *testing.T) {
	b := NewSSEBroker()
	perSlug := b.Subscribe("foo")
	defer b.Unsubscribe("foo", perSlug)
	allCh, cancel := b.SubscribeAllBoards()
	defer cancel()

	b.Publish("foo")

	select {
	case s := <-perSlug:
		if s != "foo" {
			t.Errorf("per-slug payload = %q", s)
		}
	case <-time.After(time.Second):
		t.Fatal("per-slug: no event")
	}
	select {
	case ev := <-allCh:
		if ev.Payload != "foo" {
			t.Errorf("all payload = %q", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("all: no event")
	}
}
