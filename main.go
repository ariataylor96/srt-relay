package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"

	"fmt"
	"os"
	"time"

	"github.com/chilts/sid"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RegisteredUser struct {
	gorm.Model
	UserId     string
	UserEmail  string
	ProductKey string `gorm:"index"`
	ValidUntil int64
}

const MONTH_IN_SECONDS = 2764800

var db *gorm.DB

func validateMessage(signature, body []byte) bool {
	patreonHmacKey := os.Getenv("PATREON_SECRET_KEY")
	mac := hmac.New(md5.New, []byte(patreonHmacKey))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	return subtle.ConstantTimeCompare(signature, []byte(hex.EncodeToString(expectedMAC))) == 1
}

func main() {
	server := gin.Default()
	db, _ = gorm.Open(sqlite.Open("/data.db"), &gorm.Config{})
	failure_response := gin.H{"success": false}

	db.AutoMigrate(&RegisteredUser{})

	server.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	server.POST("/patreon-callbacks", func(c *gin.Context) {
		now := time.Now().UnixNano() / 1000000
		var user RegisteredUser

		// Handle an invalid body
		rawBody, err := c.GetRawData()
		if err != nil {
			c.JSON(400, failure_response)
			return
		}

		signature := c.Request.Header.Get("X-Patreon-Signature")

		if !validateMessage([]byte(signature), rawBody) {
			fmt.Println(signature, "did not match")
			c.JSON(400, failure_response)
			return
		}

		// Handle an invalid data structure
		body := string(rawBody)
		userId := gjson.Get(body, "data.relationships.user.data.id")
		userEmail := gjson.Get(body, `included.#(type="user").attributes.email`)
		if !userId.Exists() {
			fmt.Println(userId.String(), userEmail.String())
			c.JSON(400, failure_response)
			return
		}

		// Get or create a User instance
		db.FirstOrCreate(&user, RegisteredUser{UserId: userId.String()})

		// Give them a product key if they don't have one already
		if user.ProductKey == "" {
			user.ProductKey = sid.IdBase64()
		}

		// Save their email if we have it now but didn't before
		if user.UserEmail == "" {
			user.UserEmail = userEmail.String()
		}

		// Add another month to them
		user.ValidUntil = now + (MONTH_IN_SECONDS * 1000)

		// Persist changes
		db.Model(&user).Updates(user)

		json_user, _ := json.Marshal(user)
		c.JSON(200, gin.H{"succes": true})
		fmt.Println(string(json_user))
	})

	server.Run("0.0.0.0:3000")
}
