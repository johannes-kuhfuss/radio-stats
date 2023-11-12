package app

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func basicAuth(adminuser string, adminpwhash string) gin.HandlerFunc {
	realm := "Basic realm=Authorization Required"
	return func(c *gin.Context) {
		authHeaders := c.Request.Header["Authorization"]
		found, user := checkAuth(adminuser, adminpwhash, authHeaders)
		if !found {
			c.Header("WWW-Authenticate", realm)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(gin.AuthUserKey, user)
	}
}

func checkAuth(adminuser string, adminpwhash string, authHeaders []string) (found bool, user string) {
	for _, header := range authHeaders {
		if strings.HasPrefix(header, "Basic ") {
			b64Creds := header[6:]
			creds, err := base64.StdEncoding.DecodeString(b64Creds)
			if err == nil {
				parts := strings.Split(string(creds), ":")
				if len(parts) == 2 {
					user := parts[0]
					pass := parts[1]
					if user == adminuser {
						err := bcrypt.CompareHashAndPassword([]byte(adminpwhash), []byte(pass))
						if err == nil {
							return true, user
						}
					}
				}
			}
		}
	}
	return false, ""
}
