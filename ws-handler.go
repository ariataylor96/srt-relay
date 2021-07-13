package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"
)

type User struct {
	Conn *websocket.Conn
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var listeners map[string]*[]User = make(map[string]*[]User)
var identification map[User]string = make(map[User]string)
var identifiedUsers map[string]User = make(map[string]User)
var userToRegistration map[User]RegisteredUser = make(map[User]RegisteredUser)
var lastReceived map[User]int64 = make(map[User]int64)

func removeUserFromListeners(user User, username string) {
	filtered := make([]User, 0)
	unfiltered := *listeners[username]

	for _, u := range unfiltered {
		if u != user {
			filtered = append(filtered, u)
		}
	}

	if len(filtered) == 0 {
		delete(listeners, username)
	} else {
		listeners[username] = &filtered
	}
}

func userAlreadyListening(user User, username string) bool {
	list := *listeners[username]

	for _, u := range list {
		if u == user {
			return true
		}
	}

	return false
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	user := User{conn}

	for {
		t, msg, err := conn.ReadMessage()

		if err != nil {
			break
		}

		now := time.Now().UnixNano() / 1000000
		stringified := string(msg)

		if strings.HasPrefix(stringified, "key:") {
			var registration RegisteredUser
			key := strings.Split(stringified, ":")[1]

			db.First(&registration, "product_key = ?", key)
			if registration.ID == 0 {
				conn.Close()
				return
			}

			userToRegistration[user] = registration
			defer delete(userToRegistration, user)

			fmt.Println("Fetched registration for key", key)
			continue
		}

		if strings.HasPrefix(stringified, "ident:") {
			username := strings.Split(stringified, ":")[1]

			identificationPairing, alreadyIdentified := identifiedUsers[username]

			if alreadyIdentified && identificationPairing != user {
				conn.Close()
				return
			}

			identifiedUsers[username] = user
			identification[user] = username

			defer delete(identifiedUsers, username)
			defer delete(identification, user)

			fmt.Println("Identified", user)
			fmt.Println(identification)
			continue
		}

		if strings.HasPrefix(stringified, "listen:") {
			username := strings.Split(stringified, ":")[1]
			_, ok := listeners[username]

			if !ok {
				listeners[username] = &[]User{}
			}

			if userAlreadyListening(user, username) {
				continue
			}

			*listeners[username] = append(*listeners[username], user)
			fmt.Println("Listening", user)
			fmt.Println(listeners)

			defer removeUserFromListeners(user, username)
			continue
		}

		lastMessage, hasLastMessage := lastReceived[user]

		if hasLastMessage {
			registration, isRegistered := userToRegistration[user]

			if lastMessage >= now-200 {
				if !isRegistered || registration.ValidUntil <= now {
					conn.Close()
				}
			}
		}

		lastReceived[user] = now

		username, identified := identification[user]

		if identified {
			userListeners, hasListeners := listeners[username]

			if hasListeners {
				for _, u := range *userListeners {
					u.Conn.WriteMessage(t, msg)
				}
			}
		}
	}
}
