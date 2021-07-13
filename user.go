package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"strconv"
)

type User struct {
	Conn         *websocket.Conn
	Registration RegisteredUser
	LastReceived int64
	Listeners    []*User
	Name         string
}

func (u *User) Listen(listeningUser *User) {
	u.Listeners = append(u.Listeners, listeningUser)
}

func (u *User) Unlisten(user *User) {
	filtered := make([]*User, 0)

	for _, listener := range u.Listeners {
		if listener != user {
			filtered = append(filtered, listener)
		}
	}

	u.Listeners = filtered
}

func (u *User) IsListening(user *User) bool {
	for _, listener := range u.Listeners {
		if listener == user {
			return true
		}
	}

	return false
}

func (u *User) HasValidSubscription() bool {
	return u.Registration.ValidUntil >= unixNowMs()
}

func (u *User) IsRateLimited() bool {
	// First messages are okay, and subscribers are never limited
	if u.LastReceived == 0 || u.HasValidSubscription() {
		return false
	}

	// Free users can only send so often
	freeTierBuffer, _ := strconv.ParseInt(defaultEnv("FREE_TIER_BUFFER", "200"), 10, 64)
	if u.LastReceived >= unixNowMs()-freeTierBuffer {
		return true
	}

	return false
}

func (u *User) Send(messageType int, data []byte) {
	if u.IsRateLimited() {
		u.Conn.Close()
		return
	}

	u.LastReceived = unixNowMs()
	for _, listener := range u.Listeners {
		fmt.Println(listener)
		listener.Conn.WriteMessage(messageType, data)
	}
}
