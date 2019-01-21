package wrappers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/companieshouse/document.api.ch.gov.uk/config"
	"github.com/companieshouse/document.api.ch.gov.uk/logger"
)

//Client represents an API client's stored data returned from the Accounts API
type Client struct {
	ID        string     `json:"_id"`
	Upload    bool       `json:"can_upload_documents"`
	RateLimit *Ratelimit `json:"rate_limit,omitempty"`
}

//Ratelimit represents a rate_limit subdoc within an api client document returned from the Accounts API
type Ratelimit struct {
	Window string `json:"window,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

var client http.Client

//IsAuthorized is a wrapper which checks for authorization before permitting a request
func IsAuthorized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		spew.Dump("OMG IM AUTHORIZING")
		apiClient, err := authenticate(req)
		if err != nil {
			logger.Errorf(req, "Error authenticating for client [%v]: [%s]", apiClient, err)
			if req.Body != nil {
				io.Copy(ioutil.Discard, req.Body)
				req.Body.Close()
			}
			w.WriteHeader(500)
			return
		}
		next.ServeHTTP(w, req)
		//
		//if apiClient != nil && apiClient.ID != "" {
		//	logger.Debugf(req, "Client ID authorized: [%s]", apiClient.ID)
		//	req.Header.Set("CH-Identity", apiClient.ID)
		//	if apiClient.RateLimit != nil {
		//		req.Header.Set("X-RateLimit-Limit", strconv.Itoa(apiClient.RateLimit.Limit))
		//		req.Header.Set("X-RateLimit-Window", apiClient.RateLimit.Window)
		//	}
		//	next.ServeHTTP(w, req)
		//} else {
		//	logger.Debugln(req, "Not authorized")
		//	if req.Body != nil {
		//		io.Copy(ioutil.Discard, req.Body)
		//		req.Body.Close()
		//	}
		//	w.WriteHeader(401)
		//}
	})
}

//CanUpload is a wrapper which checks for authorization and write permissions before permitting a request
func CanUpload(f func(w http.ResponseWriter, req *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		apiClient, err := authenticate(req)
		if err != nil {
			logger.Errorf(req, "Error authenticating: [%s]", err)
			if req.Body != nil {
				io.Copy(ioutil.Discard, req.Body)
				req.Body.Close()
			}
			w.WriteHeader(500)
			return
		}

		if apiClient == nil || apiClient.ID == "" {
			logger.Debugln(req, "Not authorized")
			if req.Body != nil {
				io.Copy(ioutil.Discard, req.Body)
				req.Body.Close()
			}
			w.WriteHeader(401)
		} else if apiClient.ID != "" && apiClient.Upload == true {
			logger.Debugln(req, "Client ID authorized: [%s]", apiClient.ID)
			f(w, req)
		} else if apiClient.Upload != true {
			logger.Debugln(req, "Client ID Forbidden: [%s]", apiClient.ID)
			if req.Body != nil {
				io.Copy(ioutil.Discard, req.Body)
				req.Body.Close()
			}
			w.WriteHeader(403)
		}
	}
}

//authenticate is a function which requires an authorization header
func authenticate(req *http.Request) (*Client, error) {
	cfg := config.Get()
	authHeader := req.Header.Get("Authorization")
	if len(authHeader) == 0 {
		logger.Debugln(req, "Authorization header missing")
		return nil, nil
	}

	//extract key from authHeader
	clientKey, err := splitHeader(authHeader)
	if err != nil {
		return nil, err
	}

	logger.Debugln(req, "Authenticating clientKey: [%v]", clientKey)

	//lookup key in cache
	cacheDoc, cacheHit, err := cfg.Cache.GetString(clientKey)
	if err != nil {
		logger.Warnf(req, "Cache Get error for clientKey [%s]: [%s]", clientKey, err)
		cacheHit = false
	}

	var clientDoc []byte
	//if not found in cache, call accounts site
	if !cacheHit {
		logger.Debugf(req, "Calling accounts site to verify authorization for clientKey: [%s]", clientKey)

		body, err := getAPIClient(cfg, clientKey)
		if err != nil {
			return nil, err
		}
		clientDoc = []byte(body)

		//add to cache, whether found or not
		docString := string(clientDoc)
		if len(docString) > 1 {
			cfg.Cache.SetString(clientKey, docString, true)
			logger.Debugf(req, "clientKey [%s] found", clientKey)
		} else {
			cfg.Cache.SetString(clientKey, "{}", false)
			logger.Debugf(req, "clientKey [%s] not found", clientKey)
			return nil, nil
		}
	} else {
		logger.Debugf(req, "Found authorization in cache [%s]", clientKey)
		clientDoc = []byte(cacheDoc)
	}

	logger.Debugf(req, "Decoding json for clientKey [%s]", clientKey)
	c := &Client{}

	err = json.Unmarshal(clientDoc, &c)
	return c, err
}

func getAPIClient(cfg *config.Config, clientKey string) ([]byte, error) {
	apiReq, err := http.NewRequest("GET", cfg.AccountsAPIUrl, nil)
	if err != nil {
		return nil, err
	}

	apiReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", cfg.DocAPIKey))
	apiReq.Header.Set("API-KEY-VALIDATE", clientKey)

	resp, err := client.Do(apiReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return []byte{' '}, err
	}
	return body, err
}

func splitHeader(hdr string) (string, error) {

	encClientKey := strings.SplitN(hdr, " ", 2)

	// TODO: Possibly extend client key validation.

	if encClientKey[0] != "Basic" || len(encClientKey) < 2 {
		return "", errors.New("Invalid authorization header format: " + fmt.Sprintf("%s", encClientKey))
	}

	decClientKey, err := base64.StdEncoding.DecodeString(encClientKey[1])

	if err != nil {
		return "", err
	}

	key := strings.TrimRight(string(decClientKey), ":")

	return key, err
}
