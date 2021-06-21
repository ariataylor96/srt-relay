package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
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

		stringified := string(msg)

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

func main() {
	server := gin.Default()

	server.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	server.Run("0.0.0.0:3000")
}
