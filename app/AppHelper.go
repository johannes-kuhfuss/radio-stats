// package app ties together all bits and pieces to start the program
package app

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// basicAuth checks the basic auth headers for valid credentials
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

// checkAuth validates user name and password against the configuration
func checkAuth(adminuser string, adminpwhash string, authHeaders []string) (found bool, user string) {
	for _, header := range authHeaders {
		if strings.HasPrefix(header, "Basic ") {
			parts, err := decodeBasicAuth(header)
			if err != nil {
				return false, ""
			}
			if parts[0] == adminuser {
				err := bcrypt.CompareHashAndPassword([]byte(adminpwhash), []byte(parts[1]))
				if err == nil {
					return true, user
				}
			}
		}
	}
	return false, ""
}

func decodeBasicAuth(header string) (parts []string, e error) {
	b64Creds := header[6:]
	creds, err := base64.StdEncoding.DecodeString(b64Creds)
	if err != nil {
		return nil, err
	}
	parts = strings.Split(string(creds), ":")
	if len(parts) == 2 {
		return parts, nil
	}
	return nil, errors.New("must be two parts")
}
