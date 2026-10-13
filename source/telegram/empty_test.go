package telegram

import (
	"context"
	"testing"

	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"
)

func serviceElem(u *tg.User, id int, action tg.MessageActionClass) messages.Elem {
	return messages.Elem{Msg: &tg.MessageService{ID: id, PeerID: &tg.PeerUser{UserID: u.ID}, Action: action}}
}

func TestFindEmptyChatsClassifiesHistory(t *testing.T) {
	users := []*tg.User{
		{ID: 1, FirstName: "Empty"},
		{ID: 2, FirstName: "Joined"},
		{ID: 3, FirstName: "Service"},
		{ID: 4, FirstName: "Talker"},
		{ID: 5, FirstName: "Media"},
	}
	f := &fakeFetcher{history: map[int64][]messages.Elem{}}
	for _, u := range users {
		d, _ := userDialog(u)
		f.dialogs = append(f.dialogs, d)
	}
	group := &tg.Chat{ID: 9, Title: "Group"}
	f.dialogs = append(f.dialogs, dialogs.Elem{
		Dialog: &tg.Dialog{Peer: &tg.PeerChat{ChatID: 9}}, Peer: &tg.InputPeerChat{ChatID: 9},
		Entities: entities(nil, []*tg.Chat{group}, nil),
	})
	f.history[ChatIDFromUser(2)] = []messages.Elem{serviceElem(users[1], 1, &tg.MessageActionContactSignUp{})}
	f.history[ChatIDFromUser(3)] = []messages.Elem{
		serviceElem(users[2], 2, &tg.MessageActionContactSignUp{}),
		serviceElem(users[2], 3, &tg.MessageActionHistoryClear{}),
	}
	f.history[ChatIDFromUser(4)] = []messages.Elem{
		serviceElem(users[3], 4, &tg.MessageActionContactSignUp{}),
		{Msg: &tg.Message{ID: 5, Message: "hi"}},
	}
	f.history[ChatIDFromUser(5)] = []messages.Elem{{Msg: &tg.Message{ID: 6, Media: &tg.MessageMediaPhoto{}}}}

	found, err := FindEmptyChats(context.Background(), f)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	got := map[int64]string{}
	for _, c := range found {
		got[c.Chat.ChatID] = c.Kind
	}
	want := map[int64]string{
		ChatIDFromUser(1): "empty",
		ChatIDFromUser(2): "joined_telegram",
		ChatIDFromUser(3): "service_only",
	}
	if len(got) != len(want) {
		t.Fatalf("found = %v, want %v", got, want)
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Errorf("chat %d kind = %q, want %q", id, got[id], kind)
		}
	}
}
