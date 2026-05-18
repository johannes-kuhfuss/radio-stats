package app

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestDecodeBasicAuthReturnsParts(t *testing.T) {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))

	parts, err := decodeBasicAuth(header)

	assert.Nil(t, err)
	assert.EqualValues(t, []string{"admin", "secret"}, parts)
}

func TestDecodeBasicAuthInvalidBase64ReturnsError(t *testing.T) {
	parts, err := decodeBasicAuth("Basic not-valid")

	assert.Nil(t, parts)
	assert.NotNil(t, err)
}

func TestDecodeBasicAuthWrongPartCountReturnsError(t *testing.T) {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin"))

	parts, err := decodeBasicAuth(header)

	assert.Nil(t, parts)
	assert.NotNil(t, err)
	assert.EqualValues(t, "must be two parts", err.Error())
}

func TestCheckAuthValidCredentialsReturnsUser(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))

	found, user := checkAuth("admin", string(hash), []string{header})

	assert.True(t, found)
	assert.EqualValues(t, "admin", user)
}

func TestCheckAuthInvalidPasswordReturnsFalse(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrong"))

	found, user := checkAuth("admin", string(hash), []string{header})

	assert.False(t, found)
	assert.Empty(t, user)
}

func TestBasicAuthRejectsMissingCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/private", basicAuth("admin", ""), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/private", nil)

	router.ServeHTTP(recorder, request)

	assert.EqualValues(t, http.StatusUnauthorized, recorder.Code)
	assert.EqualValues(t, "Basic realm=Authorization Required", recorder.Header().Get("WWW-Authenticate"))
}

func TestBasicAuthAcceptsValidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	router := gin.New()
	router.GET("/private", basicAuth("admin", string(hash)), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(gin.AuthUserKey))
	})
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", header)

	router.ServeHTTP(recorder, request)

	assert.EqualValues(t, http.StatusOK, recorder.Code)
	assert.EqualValues(t, "admin", recorder.Body.String())
}
