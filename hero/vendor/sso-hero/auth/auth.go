package auth 

import (
  "os"
  "encoding/json"
  "fmt"
  "net/http"
  "sso-hero/tracer"
  "net/url"
  "time"
  "crypto/tls"
  "errors"
  "encoding/base64"
)

// PIdentCredentials used to parsed the client id and secret from vcap services
type PIdentCredentials struct {
  AuthDomain string `json:"auth_domain"`
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
}

// PIdentityServiceAttr used to map to PIdentCredentials
type PIdentityServiceAttr struct {
  Creds PIdentCredentials `json:"credentials"`
}

// PIdentity struct for p-identity environment details
type PIdentity struct {
  Services [] PIdentityServiceAttr `json:"p-identity"`
}

// UAAToken stores the return token from UAA server 
type UAAToken struct {
  AccessToken string `json:"access_token"`
  TokenType string `json:"token_type"`
  RefreshToken string `json:"refresh_token"` 
  ExpiresIN int `json:"expires_in"`
  Scope string `json:"scope"`
  JTI string `json:"jti"`
}

//DecodedToken stores decoded token response from uaa check_token endpoint
type DecodedToken struct {
  UserID string `json:"user_id"`
  UserName string `json:"user_name"`
  Email string `json:"email"`
  ClientID string `json:"client_id"`
  EXP int `json:"exp"`
  Scope []string `json:"scope"`
  JTI string `json:"jti"`
  AUD []string `json:"aud"`
  SUB string `json:"sub"`
  ISS string `json:"iss"`
  IAT int `json:"iat"`
  CID string `json:"cid"`
  GrantType string `json:"grant_type"`
  AZP string `json:"azp"`
  AuthTime int `json:"auth_time"`
  ZID string `json:"zid"`
  RevSig string `json:"rev_sig"`
  Origin string `json:"origin"`
  Revocable bool `json:"revocable"`
}

// Credentials struct used to store UAA 
type Credentials struct {
  AuthDomain string `json:"auth_domain"`
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
  Token UAAToken `json:"uaatoken"`
  TokenInfo DecodedToken `json:"tokeninfo"`
  Code string `json:"code"`
  Scope string `json:"scope"`
  PIdentity *PIdentity `json:"p-identity_env"`
  AuthURL string `json:"auth_url"`
  TokenURL string `json:"token_url"`
  CheckTokenURL string `json:"check_token_url"`
  Callback string `json:"callback"`
  TraceLogs tracer.Traces `json:"tracelogs"`
}

// UAAError decodes a UAA error message
type UAAError struct {
  Err string `json:"error"`
  ErrDesc string `json:"error_description"`
}

// ErrPreventRedirect used to disable redirecting backend http client
var ErrPreventRedirect = errors.New("prevent-redirect")

// ParseIdentityEnvironment parses the p-identity environmental variable and update Credentials
// and sets up the tracer
func ParseIdentityEnvironment() (Credentials,error) {
  OauthCredentials := Credentials{}
  vcapServices := os.Getenv("VCAP_SERVICES")
  pi := new(PIdentity)
  err := json.Unmarshal([]byte(vcapServices), &pi)
  if err != nil {
    return OauthCredentials,err
  }
  if len(pi.Services) <= 0  {
    return OauthCredentials, fmt.Errorf("No P-Identity credentials found in VCAP_SERVICES")
  }
  
  envScopes := os.Getenv("SCOPE") // if variable set then we enforce required scopes
  
  OauthCredentials.Scope = envScopes
  OauthCredentials.PIdentity = pi
  OauthCredentials.AuthDomain = OauthCredentials.PIdentity.Services[0].Creds.AuthDomain
  OauthCredentials.ClientID = OauthCredentials.PIdentity.Services[0].Creds.ClientID
  OauthCredentials.ClientSecret = OauthCredentials.PIdentity.Services[0].Creds.ClientSecret
  OauthCredentials.AuthURL = "/oauth/authorize"
  OauthCredentials.TokenURL = "/oauth/token"
  OauthCredentials.CheckTokenURL = "/check_token"
  OauthCredentials.TraceLogs = tracer.NewTracer()
  return OauthCredentials, nil
} 

// StartAuthCode starts the authorization grant request and redirects user to UAA login
func(cred *Credentials) StartAuthCode(w http.ResponseWriter, r *http.Request, heroID int) {
  authorizeURL, err := url.Parse(cred.AuthDomain)
  if err != nil {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Bad Authorize URL: %s", err)))
    return
  }
  values := url.Values{}
  values.Set("response_type", "code")
  values.Set("grant_type", "authorization_code")
  values.Set("client_id", cred.ClientID)
  values.Set("redirect_uri", cred.Callback)
  
  if cred.Scope != "" { // sso will include all scopes by default
    values.Set("scope", cred.Scope)
  }
  
  values.Set("state", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("hero=%d", heroID))))
  authorizeURL.Path = cred.AuthURL
  authorizeURL.RawQuery = values.Encode()
  
  http.Redirect(w, r, authorizeURL.String(), http.StatusFound)
  return
}

// BuildImplicitURL returns the impliciit uaa url 
func (cred *Credentials) BuildImplicitURL() string  {
  implicitURL, _ := url.Parse(cred.AuthDomain)
  values := url.Values{}
  values.Set("response_type", "token")
  values.Set("client_id", cred.ClientID)
  values.Set("redirect_uri", cred.Callback)
  
  if cred.Scope != "" { // sso will include all scopes by default
    values.Set("scope", cred.Scope)
  }
  //values.Set("state", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("hero=%d", heroID))))
  implicitURL.Path = cred.AuthURL
  implicitURL.RawQuery = values.Encode()
  return implicitURL.String()
}

// GetImplicitToken extracts token from request uri
func (cred *Credentials) GetImplicitToken(w http.ResponseWriter, r *http.Request) error {
  GetImplicitTokenError := fmt.Errorf("GetImplicitTokenError")
  vals := r.URL.Query()
  accessToken := vals.Get("access_token")
  if accessToken == "" {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Missing access_token in request uri: %s", r.URL.String() )))
    return GetImplicitTokenError
  }
  token := UAAToken{}
  token.AccessToken = accessToken
  cred.Token = token 
  return nil
}

// GetAccessTokenFromCode issue the access token request from code
func (cred *Credentials) GetAccessTokenFromCode(w http.ResponseWriter, r *http.Request) error  {
  respValues := r.URL.Query()
  code := respValues.Get("code")
  GetAccessTokenFromCodeError := fmt.Errorf("GetAccessTokenFromCodeError")
  
  if code == "" {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("code not found in URL reponse")))
    return GetAccessTokenFromCodeError
  }
  cred.Code = code

  values := url.Values{}
  values.Set("response_type", "id_token")
  values.Set("grant_type", "authorization_code")
  values.Set("code", cred.Code)
  values.Set("redirect_uri", cred.Callback)

  token := UAAToken{}
  resp, err := cred.SendRequest(values, cred.TokenURL, "GET", "Basic", &token)
  if err != nil {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("TokenError:%s", err)))
    return GetAccessTokenFromCodeError
  }
  if resp.StatusCode != 200 {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("TokenError: HTTP status code: %s", err)))
    return GetAccessTokenFromCodeError
  }
  cred.Token = token
  return nil
}

// PasswordGrant requests access token using username and password
func (cred *Credentials) PasswordGrant(w http.ResponseWriter, r *http.Request, username, password string) error {
  values := url.Values{}
  values.Set("response_type", "token")
  values.Set("grant_type", "password")
  values.Set("username", username)
  values.Set("password", password)
  token := UAAToken{}
  resp, err := cred.SendRequest(values, cred.TokenURL, "POST", "Basic", &token)
  if err != nil {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("TokenError:%s", err)))
    return fmt.Errorf("PasswordGrantError")
  }
  if resp.StatusCode != 200 {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("TokenError: HTTP status code: %d", resp.StatusCode)))
    return fmt.Errorf("PasswordGrantError")
  }
  cred.Token = token
  return nil
}

// CheckToken validates token and given scope
func (cred *Credentials) CheckToken(w http.ResponseWriter, r *http.Request) error {
  
  respValues := r.URL.Query()
  scope := respValues.Get("scope")
  token := respValues.Get("token")
  
  if scope == "" {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("scope not found in URL reponse")))
    return fmt.Errorf("scope check failed")
  }
  
  if token == "" {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("token not found in URL reponse")))
    return fmt.Errorf("token check failed")
  }
  
  values := url.Values{}
  values.Set("scopes", scope)
  values.Set("token", token)
  ti := DecodedToken{}
  resp, err := cred.SendRequest(values, cred.CheckTokenURL, "POST", "Basic", &ti)
  if err != nil {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write(getJSONErrorFromString(fmt.Sprintf("check_token: %s", err)))
    return fmt.Errorf("response check failed")
  }
  if resp.StatusCode != 200 {
    uaaErr := UAAError{}
    decoder := json.NewDecoder(resp.Body)
    decoder.Decode(&uaaErr)
    return fmt.Errorf("check_token: http reponse code: %d: %s:%s", resp.StatusCode, uaaErr.Err, uaaErr.ErrDesc)
  }
  cred.TokenInfo = ti
  return nil
}

// EncodeAuth generates value for authorization header
func (cred *Credentials) EncodeAuth(context string) string {
  data := []byte(fmt.Sprintf("%s:%s", cred.ClientID, cred.ClientSecret))
  code := base64.StdEncoding.EncodeToString(data)
  return fmt.Sprintf("%s %s", context, code)
}

//SendRequest sends request
/*
   if caller is expecting request to have a response body then decode json Response
   into interface struct 
   if caller is not expecting a response body then respData will be nil
*/
func (cred *Credentials) SendRequest(values url.Values, path, reqType, authType string, respData interface{}) (*http.Response, error) {
  client := GetHTTPClient()
  nilResp := new(http.Response)
  tokenURL, err := url.Parse(cred.AuthDomain)
  if err != nil {
    return nilResp, fmt.Errorf("Bad Authorize URL:%s: %s", cred.AuthDomain, err)
  }
  tokenURL.Path = path
  tokenURL.RawQuery = values.Encode()
  
  tokenReq, err := http.NewRequest(reqType, tokenURL.String(), nil)
  if err != nil {
    return nilResp, fmt.Errorf("Unable to create request: %s", err)
  }
  
  if authType != "" {
    tokenReq.Header.Set("Authorization", cred.EncodeAuth(authType))
  }
  resp, err := client.Do(tokenReq)
  if resp.StatusCode == 200 && respData != nil {
    decoder := json.NewDecoder(resp.Body)
    derr := decoder.Decode(respData)
    if err != nil {
      return resp, fmt.Errorf("decoding response body failed: %s", derr)
    }
  }
  cred.TraceLogs.UpdateTrace(tokenReq, resp, respData)
  return resp, err
}

// GetHTTPClient build a http client that diables ssl verification and prevents http redirects
func GetHTTPClient() *http.Client {
  return &http.Client{
                CheckRedirect: func(req *http.Request, _ []*http.Request) error {
                        return ErrPreventRedirect
                },
                Timeout: 30 * time.Second,
                Transport: &http.Transport{
                        DisableKeepAlives: true,
                        TLSClientConfig: &tls.Config{
                                InsecureSkipVerify: true,
                        },
                        Proxy:               http.ProxyFromEnvironment,
                        TLSHandshakeTimeout: 10 * time.Second,
                },
        }
}

func getJSONErrorFromString(s string) []byte {
  type JError struct {
    Error string `json:"error"`
  }
  e := JError{s}
	je, _ := json.Marshal(e)
	return je
}
