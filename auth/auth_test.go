package auth 

import (
   "testing"
   "os"
   "fmt"
)

func TestParseIdentityEnvironment(t *testing.T) {
  vcapServices := `{
  "p-identity": [
   {
    "credentials": {
     "auth_domain": "https://mysso.login.system.domain.io",
     "client_id": "1e65871c-54a8-4e51-b10d-2949c9d97d82",
     "client_secret": "30bef17f-133f-420e-b506-a27b558b9f79"
    },
    "label": "p-identity",
    "name": "mysso",
    "plan": "mysso",
    "provider": null,
    "syslog_drain_url": null,
    "tags": []
   }
  ]
 }`
 os.Setenv("VCAP_SERVICES", vcapServices)
 err := ParseIdentityEnvironment()
 if err != nil {
   t.Error(err)
 }
 if len(OauthCredentials.PIdentity.Services) <= 0 {
   t.Error(fmt.Errorf("p-identity array is not populated"))
 }
}