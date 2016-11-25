package main 

import(
  "fmt"
  "net/http"
  "sso-hero/auth"
  "encoding/json"
  "os"
)

func getPowerHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  
  creds, err := auth.ParseIdentityEnvironment()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
		w.Write(getJSONErrorFromString("Could not create credentials from VCAP environment: " + err.Error()))
    return
  }
  
  err = creds.CheckToken(w,r)
  if err == nil {
    writeJSONResponse(w, creds)
    return
  }
  w.WriteHeader(http.StatusInternalServerError)
  w.Write(getJSONErrorFromString(err.Error()))
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

func main(){
  http.HandleFunc("/get.power", getPowerHandler)
  fmt.Printf("listening on port %s", os.Getenv("PORT"))
  err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
  if err != nil {
    fmt.Printf("Failed to start http server: %s\n", err)
  }
}
