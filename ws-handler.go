package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var users []User = make([]User, 0)
var nameToUser map[string]*User = make(map[string]*User)

func removeUserFromList(user User) {
	filtered := make([]User, 0)

	for _, u := range users {
		if &u != &user {
			filtered = append(filtered, u)
		}
	}

	users = filtered
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	user := User{Conn: conn, Listeners: make([]*User, 0)}

	users = append(users, user)
	defer removeUserFromList(user)

	for {
		t, msg, err := conn.ReadMessage()

		if err != nil {
			break
		}

		stringified := string(msg)

		if strings.HasPrefix(stringified, "key:") {
			var registration RegisteredUser
			key := strings.Split(stringified, ":")[1]

			db.First(&registration, "product_key = ?", key)
			if registration.ID == 0 {
				conn.Close()
				return
			}

			user.Registration = registration

			fmt.Println("Fetched registration for key", key)
			continue
		}

		if strings.HasPrefix(stringified, "ident:") {
			username := strings.Split(stringified, ":")[1]

			identificationPairing, alreadyIdentified := nameToUser[username]

			if alreadyIdentified && identificationPairing != &user {
				conn.Close()
				return
			}

			user.Name = username

			nameToUser[username] = &user
			defer delete(nameToUser, username)

			fmt.Println("Identified", username)
			continue
		}

		if strings.HasPrefix(stringified, "listen:") {
			username := strings.Split(stringified, ":")[1]
			userToListenTo, ok := nameToUser[username]

			if !ok || userToListenTo.IsListening(user) {
				continue
			}

			userToListenTo.Listen(&user)

			if user.Name != "" {
				fmt.Println("LISTEN:" + user.Name + ":" + userToListenTo.Name)
			} else {
				fmt.Println("LISTEN" + userToListenTo.Name + ":")
			}

			defer userToListenTo.Unlisten(&user)
			continue
		}

		if user.IsRateLimited() {
			conn.Close()
			return
		}

		user.Send(t, msg)
	}
}
