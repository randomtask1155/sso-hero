
var GetHeroAuthURL = 'authorization_code';
var GetImplicitURL = "implicit";
var SetImplicitURL = 'set_implicit';
var GetHeroPasswordURL = 'password';
var CheckPowerURL = 'powers.check';
var GetWebAppCredsURL = 'creds.get';

var Powers = {
  "fly": {
    "name": "fly",
    "scope": "power.fly",
    "audio": "a-herofly",
    "image": '<img id="hero-image" width="150" src="img/hero-fly.gif"></img>',
    "enabled": false
  },
  "strength": {
    "name": "strength",
    "scope": "power.strength",
    "audio": "a-herostrength",
    "image": '<img id="hero-image" width="100" src="img/hero-strength.png"></img>',
    "enabled": false
  },
  "invisibility": {
    "name": "invisibility",
    "scope": "power.invisibility",
    "audio": "a-heroinvisible",
    "image": '<img id="hero-image" width="100" src="img/hero-invisibility.png"></img>',
    "enabled": false
  }
};

var JavascriptToken = {
  "access_token": "",
  "expires_in": 0,
  "scopes": "",
  "jti": ""
};

var NoScopeSound = 'a-noscope';
var PortalSound = 'a-portal';
var webType = "authorization_code";
var mobileType = "password";
var javascriptType = "implicit";
var startType = "start";
var HeroID = 0;
var webAppCredentials = {};

/*
  check url for hero id and set global var if found
*/
function CheckURLParams() {
  var params={};
  window.location.search.replace(/[?&]+([^=&]+)=([^&]*)/gi,function(str,key,value){params[key] = value;});
  if ( params['state'] ) {
    encodedstring = params['state'].replace('%3D', '=');
    var decodedstring =  atob(encodedstring);
    var kvs = decodedstring.split("?");
    var pmap = {};
      for ( i = 0; i < kvs.length; i++) {
        var kv = kvs[i].split("=");
        pmap[kv[0]] = kv[1];
      }
      params = pmap;
    }
  if ( params['hero'] ) {
    HeroID = params['hero'];
  }
  
  var hashParams = {};
  var hashArray = window.location.hash.replace('#', '').split('&');
  var hashParams = {};
  for ( var i = 0; i < hashArray.length; i++) {
	   var kv = hashArray[i].split('=');
     if (kv.length >= 2 ) {
  	    hashParams[kv[0]] = kv[1];
      }
  } 
  
  if ( hashParams.access_token ) {
    JavascriptToken.access_token = hashParams.access_token;
    JavascriptToken.expires_in = hashParams.expires_in;
    console.log(hashParams.scope);
    console.log(hashParams.scope.replace(/\%20/g, ' '));
    JavascriptToken.scopes = hashParams.scope.replace(/\%20/g, ' ');
    JavascriptToken.jti = hashParams.jti;
  }
    
}

/*
  update html for div id #NAVBAR 
*/
function GenNavBar() {
  var nav = '<nav class="navbar navbar-inverse center">' +
    '<div class="collapse navbar-collapse navbar-inner" id="bs-example-navbar-collapse-1">' +
      '<ul class="nav navbar-nav">' +
        '<li class="active"><a href="https://github.com/randomtask1155/sso-hero" target="_blank">github</a></li>' +
    '</div>' +
  '</nav>';
  
  $('#NAVBAR').html(nav);
}

var HeroMainView = '<div class="container">' +
  '<div class="row powerrow">' +
    '<div class="col-sm-6">' +
      '<div id="hero-image"><img id="hero-image" width="100" src="img/hero-figure.gif"></img></div>'  +
    '</div> <!-- end column -->' +
    
    '<div class="col-sm-1 powercol"><center>' +
    
      '<div class="panel panel-info">' +
        '<div class="panel-heading"><h3 class="panel-title">power.fly</h3></div>' +
        '<a href="#" onclick="CheckPower(Powers.fly);"><div class="panel-body"><span id="power-fly-glyph" class="glyphicon glyphicon-remove"></span> - fly</div></a>' +
      '</div>' +
      
    '</div></center> <!-- end column -->' +
    
    '<div class="col-sm-1 powercol"><center>' +
      
    '<div class="panel panel-info">' +
      '<div class="panel-heading"><h3 class="panel-title">power.strength</h3></div>' +
      '<a href="#" onclick="CheckPower(Powers.strength);"><div class="panel-body"><span id="power-strength-glyph" class="glyphicon glyphicon-remove"></span> - strength</div></a>' +
    '</div>' +
      
    '</div></center> <!-- end column -->' +
    
    '<div class="col-sm-1 powercol"><center>' +
      
    '<div class="panel panel-info">' +
      '<div class="panel-heading"><h3 class="panel-title">power.invisibility</h3></div>' +
      '<a href="#" onclick="CheckPower(Powers.invisibility);"><div class="panel-body"><span id="power-invisibility-glyph" class="glyphicon glyphicon-remove"></span> - invisibility</div></a>' +
    '</div>' +
      
    '</div></center> <!-- end column -->' +
    
   '</div> <!-- end row -- >' +
   '</div> <!-- end container -->';

   var tracersView = '<div class="col-xs-6">' +
        '<div class="panel panel-danger">' +
          '<div class="panel-heading"><h3 class="panel-title">Application Trace</h3></div>' +
          '<div id="appTrace"><p>...</p></div>' +
        '</div>' +
      '</div>  <!-- end column -->' +
      
      '<div class="col-xs-6">' +
        '<div class="panel panel-danger">' +
          '<div class="panel-heading"><h3 class="panel-title">Hero Server Trace</h3></div>' +
          '<div id="heroTrace"><p>...</p></div>' +
        '</div>' +
      '</div> <!-- end column -->';

function PrintMainView() {
  $('#DisplayGrantType').html('<p><h1>Current Grant Type is <strong>' + GrantType + '</strong></h1></p>')
  $('#mainView').html(HeroMainView);
  $('#Tracers').html(tracersView);
  $('#hero-control-panel').show();
  
  if (AUTHENTICATED_TYPE == webType) {
    getWebAppCredentials();
  } 
  if (AUTHENTICATED_TYPE == javascriptType) {
    $.ajax({
      url: SetImplicitURL,
      type: 'post',
      dataType: 'json',
      success: function (data) {
        HeroID = data.hero_id;
        getWebAppCredentials();
      },
      error: function(data) {
        alert(JSON.stringify(data.responseJSON));
      },
      data: JSON.stringify(JavascriptToken)
     });
  }
 }
 
 function Authenticate() {
   console.log(GrantType);
   console.log(webType);
   if (GrantType == webType) {
     GetAuthorizeCode();
   } else if ( GrantType == mobileType ) {
     $('#CredentialsModal').modal('show');
   } else if ( GrantType == javascriptType) {
     $.ajax({
       url: GetImplicitURL,
       type: 'get',
       dataType: 'json',
       success: function (data) {
         window.location.href = data.implicit_url;
       },
       error: function(data) {
         alert(JSON.stringify(data.responseJSON));
       }
      });
   }
 }
 
 function PlayAudioFile(id) {
   document.getElementById(id).play();
 }
 
 function GetAuthorizeCode() {
   window.location.href = GetHeroAuthURL + "?authtype=" + GrantType;
 }
 
function PasswordAuthExecute(){
  $('#CredentialsModal').modal('hide');
  $.ajax({
   url: GetHeroPasswordURL + "?username=" + $('#formUsername').val() + "&password=" + $('#formPassword').val(),
   type: 'get',
   dataType: 'json',
   success: function (data) {
     HeroID = data.heroid;
     AUTHENTICATED_TYPE = GrantType; // mark state as authenticated.. not really required but cleaner
     getWebAppCredentials();
   },
   error: function(data) {
     alert(JSON.stringify(data.responseJSON));
   }
 });
}
 
 function getWebAppCredentials() {
   $.ajax({
    url: GetWebAppCredsURL + "?hero=" + HeroID,
    type: 'get',
    dataType: 'json',
    success: function (data) {
       $('#appTrace').html('<ul>' +
         '<li>Auth Domain: <strong>' + data.auth_domain + '</strong></li>' +
        '<li>Client ID: <strong>' + data.client_id + '</strong></li>' +
        '<li>Client Secret: <strong>' + data.client_secret + '</strong></li>' +
        '<li>Authorized Grant Code: <strong>' + data.code + '</strong></li>' +
        '<li>Granted UAA Token</br>' +
          '<pre>' + JSON.stringify(data.uaatoken, null, 2) + '</pre></li>' +
        '<li>Http Request logs</br>' +
          '<pre>' + JSON.stringify(data.tracelogs, null, 2) + '</pre></li>' +
        '</ul>');
       PlayAudioFile(PortalSound);  
       
       webAppCredentials = data;
       scopes = data.uaatoken.scope.split(" ");
       for (var i = 0;i <= scopes.length; i++) {
         if (!scopes[i]) { // protect form null strings
           break;
         }
         var rp = scopes[i].split("."); 
         if ( rp.length == 2 ){
           var r = rp[0];
           var p = rp[1];
         } else {
           console.log("unable to parse scope: " + scopes[i]);
         }
         if ( Powers[p] ) {
           Powers[p].enabled = true;
           $('#power-' + p + '-glyph').removeClass("glyphicon glyphicon-remove").addClass("glyphicon glyphicon-ok");
         }
       }
    },
    error: function(data) {
      PlayAudioFile('a-noscope');
      alert(JSON.stringify(data.responseJSON));
    }
  });
 }
 
 // verifies app has access to scope and returns credentials for display of traces
 function CheckPower(power) {   
   $.ajax({
    url: CheckPowerURL + "?scope=" + power.scope + "&hero=" + HeroID,
    type: 'get',
    dataType: 'json',
    success: function (data) {
      $('#heroTrace').fadeOut();
      //$('#heroTrace').html("<pre>" + JSON.stringify(data.herocreds, null, 2) + "</pre>");
      
      $('#heroTrace').html('<ul>' +
        '<li>Auth Domain: <strong>' + data.herocreds.auth_domain + '</strong></li>' +
       '<li>Client ID: <strong>' + data.herocreds.client_id + '</strong></li>' +
       '<li>Client Secret: <strong>' + data.herocreds.client_secret + '</strong></li>' +
       '<li>Token info - Basic user infromation</br>' +
         '<pre>' + JSON.stringify(data.herocreds.tokeninfo, null, 2) + '</pre></li>' +
       '<li>Http Request logs between sso-hero app and hero resource server</br>' +
         '<pre>' + JSON.stringify(data.tracelogs, null, 2) + '</pre></li>' +
       '<li>Http Request logs between hero server and UAA</br>' +
         '<pre>' + JSON.stringify(data.herocreds.tracelogs, null, 2) + '</pre></li>' +
       '</ul>');
      
      $('#heroTrace').fadeIn();
      $('#hero-image').html(power.image);
       PlayAudioFile(power.audio);
    },
    error: function(data) {
      PlayAudioFile(NoScopeSound);
      alert(JSON.stringify(data.responseJSON));
    }
  });
 }