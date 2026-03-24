package whatsapp

import (
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// shouldHandle returns true if this message should be routed to the channel.
func shouldHandle(senderJID, ownerJID string, isFromMe, isGroup bool, text string) bool {
	if isFromMe || isGroup {
		return false
	}
	if strings.TrimSpace(text) == "" {
		return false
	}
	// Compare the user part of the JID (before @).
	return jidUser(senderJID) == jidUser(ownerJID)
}

// jidUser extracts the user part from a JID (e.g. "1234@s.whatsapp.net" -> "1234").
func jidUser(jid string) string {
	if user, _, ok := strings.Cut(jid, "@"); ok {
		return user
	}
	return jid
}

// RegisterEventHandler registers a whatsmeow event handler that routes
// messages from the owner to the channel.
func RegisterEventHandler(wmClient *whatsmeow.Client, ch *Channel, ownerJID string) {
	wmClient.AddEventHandler(func(evt any) {
		msg, ok := evt.(*events.Message)
		if !ok {
			return
		}

		senderJID := msg.Info.Sender.String()
		isFromMe := msg.Info.IsFromMe
		isGroup := msg.Info.IsGroup

		text := extractText(msg)
		if !shouldHandle(senderJID, ownerJID, isFromMe, isGroup, text) {
			return
		}

		ch.HandleIncoming(text, senderJID, msg.Info.ID)
	})
}

func extractText(msg *events.Message) string {
	if msg.Message == nil {
		return ""
	}
	if conv := msg.Message.GetConversation(); conv != "" {
		return conv
	}
	if ext := msg.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
