package main 

import (
  "net/http"
  "html/template"
  "os"
  "fmt"
  "sso-hero/auth"
  "sso-hero/tracer"
  "encoding/json"
  "sync"
  "time"
  "strconv"
  "strings"
  "encoding/base64"
  "net/url"
  "io/ioutil"
)

// VCAPAppDetails stores details from VCAP_APPLICATION environment variable
type VCAPAppDetails struct {
  AppID string `json:"application_id"`
  AppName string `json:"application_name"`
  AppURI []string `json:"application_uris"`
}

var (
  startPageTemplate = template.Must(template.ParseFiles("tmpl/index.tmpl")) 
  autherrorPageTemplate = template.Must(template.ParseFiles("tmpl/autherror.tmpl")) 
  // AppDetails globaly set app details
  AppDetails = VCAPAppDetails{}
  
  // HeroCache sotres all the active hero ids and their credentails 
  HeroCache map[int]Hero
  idCounter = 0 // keep track of hero ID
  mutex = &sync.Mutex{} // make sure only one routine accesses idCounter at a time
  
  // HeroServerURL is the address for the hero app instance 
  HeroServerURL string
  
  // GrantType dictaes how the app should behave with sso 
  GrantType string
  
  // define the support grant types
  webApp = "authorization_code"
  mobileApp = "password"
  javascriptApp = "implicit"
)

// AuthTypes used to build the template for index.html
type AuthTypes struct {
  AuthenticatedType string 
  AuthType string
}

// Hero struct inculde the information about the 
type Hero struct {
  ID int `json:"id"`
  Creds *auth.Credentials `json:"credntials"`
  Expires time.Time `json:"expires"` // Date when hero id will expire
}

func newHero() (int, error) {
  h := Hero{}
  h.generateHeroID()
  creds, err := auth.ParseIdentityEnvironment()
  if err != nil {
    return 0, fmt.Errorf("Could not get identity credentails: %s", err)
  }
  h.Creds = &creds
  h.Creds.Callback = fmt.Sprintf("https://%s/authorization_callback", AppDetails.AppURI[0])
  h.Expires = time.Now().Add(time.Duration(3600) * time.Second) // expires in 1 hour
  h.addToCache()
  return h.ID, nil
}

func getHeroCreds(heroID int) *auth.Credentials {
  _,ok := HeroCache[heroID]
  if !ok {
    return &auth.Credentials{}
  } 
  return HeroCache[heroID].Creds
}

// Once hero expires flush it from memory
func rotateHeroCredCache() {
  for {
    for _,h := range HeroCache {
      if time.Now().After(h.Expires) {
        h.removeFromCache()
      }
    }
    time.Sleep(time.Duration(60) * time.Second)
  }
}

func(h *Hero) generateHeroID() {
  mutex.Lock()
  idCounter++
  h.ID = idCounter
  mutex.Unlock()
  return 
}

func (h *Hero) addToCache() {
  mutex.Lock()
  HeroCache[h.ID] = *h
  mutex.Unlock()
}

func (h *Hero) removeFromCache() {
  mutex.Lock()
  delete(HeroCache, h.ID)
  mutex.Unlock()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  startPageTemplate.Execute(w, AuthTypes{"", GrantType})
}

func webRootHandler(w http.ResponseWriter, r *http.Request) {
  startPageTemplate.Execute(w, AuthTypes{webApp, GrantType})
}

func javascriptRootHandler(w http.ResponseWriter, r *http.Request) {
  startPageTemplate.Execute(w, AuthTypes{javascriptApp, GrantType})
}

// TODO handl error response 
// https://sso-hero.cfapps-07.haas-59.pez.pivotal.io/authorize_code?error=invalid_scope&error_description=Invalid%20scope:%20power.invisibility.%20Did%20you%20know%20that%20you%20can%20get%20default%20requested%20scopes%20by%20simply%20sending%20no%20value?&scope=power.fly%20openid
func authorizeCallbackHandler(w http.ResponseWriter, r *http.Request){
  
  // if the UAA server return error then display autherror page
  vals := r.URL.Query()
  if vals.Get("error") != "" {
    autherrorPageTemplate.Execute(w,fmt.Sprintf("Error: %s\nDescription: %s", vals.Get("error"), vals.Get("error_description")))
    return
  }
  if vals.Get("state") == "" && GrantType == webApp {
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("authorizeCallbackHandler: Could not find \"state=\" in request URL")))
    return
  } else if GrantType == webApp {
    // this should be properlly paramitized but this only for example so its ok to assume there will be only one param passed in through state
    stateString, err := base64.StdEncoding.DecodeString(vals.Get("state"))
    if err != nil {
      w.WriteHeader(http.StatusBadRequest)
      w.Write(getJSONErrorFromString(fmt.Sprintf("unable to get state information from %s: %s\n", vals.Get("state"), err)))
      return
    }
    stateHeroID := strings.Split(fmt.Sprintf("%s", stateString), "=")
    if len(stateHeroID) != 2 {
      w.WriteHeader(http.StatusBadRequest)
      w.Write(getJSONErrorFromString(fmt.Sprintf("State Information does not have valid data.. expecting \"hero=<hero id>\": %s", stateString)))
      return
    }
    hID, _ := strconv.Atoi(stateHeroID[1])

    c := getHeroCreds(hID)
    err = c.GetAccessTokenFromCode(w,r)
    if err != nil {
      return
    }
    http.Redirect(w, r, fmt.Sprintf("/web?hero=%s", stateHeroID[1]), http.StatusFound)
  }
  startPageTemplate.Execute(w, AuthTypes{javascriptApp, GrantType})
}

func getCredentialsHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  
  vals := r.URL.Query()
  if vals.Get("hero") == "" {
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Could not find \"hero=\" in request URL")))
    return
  }
  hID,_ := strconv.Atoi(vals.Get("hero")) 
  writeJSONResponse(w,getHeroCreds(hID))
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {    
  heroID, err := newHero()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", err)))
    return
  }
  c := getHeroCreds(heroID)
  c.StartAuthCode(w,r, heroID)
}

func passwordHandler(w http.ResponseWriter, r *http.Request) { 
  vals := r.URL.Query()
  if vals.Get("username") == "" || vals.Get("password") == "" { 
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Missing url parameter \"username=\" or \"password=\" in request URL")))
    return
  }
  heroID, err := newHero()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", err)))
    return
  }
  c := getHeroCreds(heroID)
  err = c.PasswordGrant(w,r, vals.Get("username"), vals.Get("password"))
  if err != nil {
    return
  }
  type PassResp struct {
    HeroID string `json:"heroid"`
  }
  resp := PassResp{fmt.Sprintf("%d", heroID)}
  writeJSONResponse(w, resp)
}

func implicitHandler(w http.ResponseWriter, r *http.Request) {  
  heroID, err := newHero()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", err)))
    return
  }
  cred := getHeroCreds(heroID)
  type ImpResp struct {
    URL string `json:"implicit_url"`
  } 
  writeJSONResponse(w, ImpResp{cred.BuildImplicitURL()})
  return 
}

func setImplicitHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  heroID, err := newHero()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", err)))
    return
  }
  cred := getHeroCreds(heroID)
  
  type JSToken struct {
    AccessToken string `json:"access_token"`
    ExpiresIN string `json:"expires_in"`
    Scope string `json:"scopes"`
    JTI string `json:"jti"`
  }
  jst := JSToken{}
  decoder := json.NewDecoder(r.Body)
  err = decoder.Decode(&jst)
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString("decoder error " + err.Error()))
    return
  }
  
  expInt, _ := strconv.Atoi(jst.ExpiresIN)
  uaaToken := auth.UAAToken{}
  uaaToken.AccessToken = jst.AccessToken 
  uaaToken.ExpiresIN = expInt
  uaaToken.Scope = jst.Scope 
  uaaToken.JTI = jst.JTI
  cred.Token = uaaToken
  
  type RespHero struct {
    HeroID int `json:"hero_id"`
  }
  writeJSONResponse(w, RespHero{heroID})
}

/*
  Send checkToken request to hero server with scope and access token
*/
func powerCheckHandler(w http.ResponseWriter, r *http.Request) {
  vals := r.URL.Query()
  scope := vals.Get("scope")
  heroID := vals.Get("hero")
  if scope == "" {
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Unable to get scope from url: %s", r.URL.String())))
    return
  }
  if heroID == "" {
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Unable to get hero id from url: %s", r.URL.String())))
    return
  }
  
  hid,_ := strconv.Atoi(heroID)
  c := getHeroCreds(hid)
  if c.Token.AccessToken == "" {
    w.WriteHeader(http.StatusBadRequest)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Hero with ID %s is missing access token\nHero ID's expire after one hour so you should refresh the page and try again.  You can run this command to check the existing credentials:\ncurl http://sso-hero.domain/creds.get?hero=%s", heroID, heroID)))
    return
  }
  
  if HeroServerURL == "" {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("HEROSERVER environment variable is not set")))
    return
  }
  
  // build http request to send to hero server  
  client := auth.GetHTTPClient()
  checkURL, err := url.Parse(HeroServerURL)
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Could not parse hero server url. Please make sure HEROSERVER env is set to a valid URL: %s", err)))
    return
  }
  values := url.Values{}
  values.Set("scope", scope)
  values.Set("token", c.Token.AccessToken)
  checkURL.Path = "get.power"
  checkURL.RawQuery = values.Encode()
  
  checkReq, err := http.NewRequest("GET", checkURL.String(), nil)
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Could not create request: %s", err)))
    return
  }
  
  resp, err := client.Do(checkReq) 
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", err)))
    return
  }
  if resp.StatusCode != 200 {
    body, _ := ioutil.ReadAll(resp.Body)
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("%s", body)))
    return
  }
  heroCreds := auth.Credentials{}
  decoder := json.NewDecoder(resp.Body)
  derr := decoder.Decode(&heroCreds)
  if derr != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(getJSONErrorFromString(fmt.Sprintf("Could not decode hero server response: %s", derr)))
    return
  }
  traceLogs := tracer.NewTracer()
  traceLogs.UpdateTrace(checkReq, resp, heroCreds)
  type CheckResp struct {
    HeroCreds auth.Credentials `json:"herocreds"`
    TraceLogs tracer.Traces `json:"tracelogs"`
  }
  powerResp := CheckResp{heroCreds, traceLogs}
  writeJSONResponse(w, powerResp)
}

func getJSONErrorFromString(s string) []byte {
  type JError struct {
    Error string `json:"error"`
  }
  e := JError{s}
	je, _ := json.Marshal(e)
	return je
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	jdata, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(getJSONErrorFromString("Failed to marshal data: " + err.Error()))
		return
	}
	w.Write(jdata)
	return
}

// Try to get app details from VCAP_APPLICATION.  If that fails then default to http://127.0.0.1:PORT
func setAppDetails() {
  vcapApp := os.Getenv("VCAP_APPLICATION")
  if vcapApp == "" {
    AppDetails.AppURI = []string{fmt.Sprintf("http://127.0.0.1:%s", os.Getenv("PORT"))}
    return 
  }
  err := json.Unmarshal([]byte(vcapApp), &AppDetails)
  if err != nil {
    fmt.Printf("Unmarshalling VCAP_APPLICATION env variable failed: %s\n", err)
    fmt.Printf("reverting to default http://127.0.0.1:%s\n", os.Getenv("PORT"))
    AppDetails.AppURI = []string{fmt.Sprintf("http://127.0.0.1:%s", os.Getenv("PORT"))}
  }
}

func main() {
  HeroServerURL = os.Getenv("HEROSERVER")
  GrantType = os.Getenv("GRANT_TYPE")
  setAppDetails()
  HeroCache = make(map[int]Hero)
  go rotateHeroCredCache()
  /*_, err := auth.ParseIdentityEnvironment() // make sure we can parse the vacp environment
  if err != nil {
    fmt.Printf("Couldn not parse p-identity in VCAP_SERVICES environment variable: %s\n", err)
  }
  auth.OauthCredentials.Callback = fmt.Sprintf("https://%s/authorize_code", AppDetails.AppURI[0])*/
  
  // start http services 
  http.HandleFunc("/", rootHandler)
  http.HandleFunc("/web", webRootHandler)
  http.HandleFunc("/javascript", javascriptRootHandler)
  http.HandleFunc("/authorization_code", authorizeHandler)
  http.HandleFunc("/password", passwordHandler)
  http.HandleFunc("/implicit", implicitHandler)
  http.HandleFunc("/set_implicit", setImplicitHandler)
  http.HandleFunc("/powers.check", powerCheckHandler)
  http.HandleFunc("/creds.get", getCredentialsHandler)
  http.HandleFunc("/authorization_callback", authorizeCallbackHandler)
	http.Handle("/img/", http.FileServer(http.Dir("")))
	http.Handle("/js/", http.FileServer(http.Dir("")))
	http.Handle("/css/", http.FileServer(http.Dir("")))
  http.Handle("/sound/", http.FileServer(http.Dir("")))
  http.Handle("/fonts/", http.FileServer(http.Dir("")))
  fmt.Printf("listening on port %s\n", os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		fmt.Printf("Failed to start http server: %s\n", err)
	}
  
}