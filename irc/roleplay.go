// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package irc

import (
	"fmt"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/oragono/oragono/irc/caps"
)

const (
	npcNickMask   = "*%s*!%s@npc.fakeuser.invalid"
	sceneNickMask = "=Scene=!%s@npc.fakeuser.invalid"
)

// SCENE <target> <text to be sent>
func sceneHandler(server *Server, client *Client, msg ircmsg.IrcMessage) bool {
	target := msg.Params[0]
	message := msg.Params[1]
	sourceString := fmt.Sprintf(sceneNickMask, client.nick)

	sendRoleplayMessage(server, client, sourceString, target, false, message)

	return false
}

// NPC <target> <sourcenick> <text to be sent>
func npcHandler(server *Server, client *Client, msg ircmsg.IrcMessage) bool {
	target := msg.Params[0]
	fakeSource := msg.Params[1]
	message := msg.Params[2]

	_, err := CasefoldName(fakeSource)
	if err != nil {
		client.Send(nil, client.server.name, ERR_CANNOTSENDRP, target, client.t("Fake source must be a valid nickname"))
		return false
	}

	sourceString := fmt.Sprintf(npcNickMask, fakeSource, client.nick)

	sendRoleplayMessage(server, client, sourceString, target, false, message)

	return false
}

// NPCA <target> <sourcenick> <text to be sent>
func npcaHandler(server *Server, client *Client, msg ircmsg.IrcMessage) bool {
	target := msg.Params[0]
	fakeSource := msg.Params[1]
	message := msg.Params[2]
	sourceString := fmt.Sprintf(npcNickMask, fakeSource, client.nick)

	_, err := CasefoldName(fakeSource)
	if err != nil {
		client.Send(nil, client.server.name, ERR_CANNOTSENDRP, target, client.t("Fake source must be a valid nickname"))
		return false
	}

	sendRoleplayMessage(server, client, sourceString, target, true, message)

	return false
}

func sendRoleplayMessage(server *Server, client *Client, source string, targetString string, isAction bool, message string) {
	if isAction {
		message = fmt.Sprintf("\x01ACTION %s (%s)\x01", message, client.nick)
	} else {
		message = fmt.Sprintf("%s (%s)", message, client.nick)
	}

	target, cerr := CasefoldChannel(targetString)
	if cerr == nil {
		channel := server.channels.Get(target)
		if channel == nil {
			client.Send(nil, server.name, ERR_NOSUCHCHANNEL, client.nick, targetString, client.t("No such channel"))
			return
		}

		if !channel.CanSpeak(client) {
			client.Send(nil, client.server.name, ERR_CANNOTSENDTOCHAN, channel.name, client.t("Cannot send to channel"))
			return
		}

		if !channel.flags[ChanRoleplaying] {
			client.Send(nil, client.server.name, ERR_CANNOTSENDRP, channel.name, client.t("Channel doesn't have roleplaying mode available"))
			return
		}

		for _, member := range channel.Members() {
			if member == client && !client.capabilities.Has(caps.EchoMessage) {
				continue
			}
			member.Send(nil, source, "PRIVMSG", channel.name, message)
		}
	} else {
		target, err := CasefoldName(targetString)
		user := server.clients.Get(target)
		if err != nil || user == nil {
			client.Send(nil, server.name, ERR_NOSUCHNICK, client.nick, target, client.t("No such nick"))
			return
		}

		if !user.flags[UserRoleplaying] {
			client.Send(nil, client.server.name, ERR_CANNOTSENDRP, user.nick, client.t("User doesn't have roleplaying mode enabled"))
			return
		}

		user.Send(nil, source, "PRIVMSG", user.nick, message)
		if client.capabilities.Has(caps.EchoMessage) {
			client.Send(nil, source, "PRIVMSG", user.nick, message)
		}
		if user.flags[Away] {
			//TODO(dan): possibly implement cooldown of away notifications to users
			client.Send(nil, server.name, RPL_AWAY, user.nick, user.awayMessage)
		}
	}
}
