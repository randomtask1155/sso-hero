
# How to setup
## Create a new sso instance 

- Manage service plans via https://p-identity.systemdomain and make sure your org can use this service 

## Create new sso resource 

- connect to your new sso instances dashboard by going to apps manager -> org -> serivces -> sso instance -> manage

The `power` resource should have the following permissions granting the frontend apps access to your super powers

- power.fly 
- power.strength 
- power.invisibility

## Update the `manifest.yml`

Edit environment variable `HEROSERVER` with your app domain.  This allows the frontend apps to find the hero server resource.  Also feel free to change the appname from `hero` to what ever you like, but make sure this url is updated accordingly.

```
env:
  HEROSERVER: http://hero.<APP DOMAIN>
```

## Push All of the apps using `./manifest.yml`

```
cf push -f manifest.yml
```

Frontend Apps: 
- sso-hero-web 
  - Uses `authoriation_code` grant type
- sso-hero-mobile 
  - Uses `password` grant type
- sso-her-implicit 
  - Uses `implicit` grant type
  
Backend App:
- hero 
  - Uses `client_credentials` and will verify the given token has permissions to access the request resource

## Add scopes to each of the front end apps

- From SSO management dashboard select the each of the front end apps
- Assign any scopes you want from the power resource and feel free to assign them all or only a few if you want to see how things fail

# for the `sso-hero-web` app only add the callback redirect URI

this uri is used when users browser is redirected from the `sso-hero-web` app to your sso login page.  Once the authorization grant code is created sso will redirect the browser back to `sso-hero-web` using the `/authorization_code` endpoint

- https://sso-hero-web.<app domain>/authorization_callback

# Create new users accounts in your sso instance
- PCF 1.8 [Doc Link](http://docs.pivotal.io/p-identity/1-8/configure-apps/index.html#admin)
- PCF 1.7 [Doc Link](http://docs.pivotal.io/p-identity/1-7/configure-id-providers.html#add-to-int)

the following examples creates user1 and user2

```
uaac user add --emails user1@domain.com
User name:  user1
Password:  
Verify password:  
user account successfully added

uaac user add --emails user2@domain.com
User name:  user2
Password:  
Verify password:  
user account successfully added
```

Then we need to set the their scopes 

```
uaac member add power.fly user1
uaac member add power.strength user1
uaac member add power.invisibility user1

uaac member add power.fly user2
uaac member add power.strength user2
uaac member add power.invisibility user2
```


# Build notes 
[godeps](https://github.com/tools/godep) is used to manage dependencies between both apps. Currently `auth` and `tracer` packages are organized by godeps.  If you plan to make changes to these packages you will have to regenerate the godeps dependency references.
- go get github.com/tools/godep
- rm -rf hero/Godeps hero/vendor web/Godeps web/vendor 
- cd hero; godep save;
- cd web; godep save;



