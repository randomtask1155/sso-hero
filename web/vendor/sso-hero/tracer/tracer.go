package tracer 

import (
  "net/http"
  "net/url"
  "mime/multipart"
)

var (
  // Tracer is global var that can be used to store all http requests and responses
  Tracer = Traces{}
)

// TraceRequest converts data from http.Request so it can be marshalled as json
type TraceRequest struct {
  Method string `json:"method"`
  URL *url.URL `json:"url"`
  Proto string `json:"proto"`
  Header http.Header `json:"header"`
  //Body string `json:"body"`
  ContentLength int64  `json:"contentlength"`
  Host string `json:"host"`
  Form url.Values `json:"Form"`
  PostForm url.Values `json:"PostForm"`
  MultipartForm *multipart.Form `json:"multipartform"`
  Trailer http.Header `json:"trailer"`
  RemoteAddr string `json:"remoteaddr"`
  RequestURI string `json:"requesturi"`
}

// TraceResponse converts data from http.Response to it can be marhsalled as json
type TraceResponse struct {
  Status string `json:"status"`
  StatusCode int `json:"statuscode"`
  Proto string `json:"proto"`
  Header http.Header `json:"header"`
  ContentLength int64 `json:"contentlength"`
  TransferEncoding []string `json:"transferencoding"`
  Trailer http.Header`json:"trailer"`
  Body interface{} `json:"body"`
}

// TraceInfo stores both a request and resonse
type TraceInfo struct {
  Request TraceRequest `json:"request"`
  Response TraceResponse `json:"response"`
}

// Traces is used to store the http requests and responses
type Traces struct {
  Trace []TraceInfo `json:"trace"` 
}

// NewTracer create a new empty tracer struct
func NewTracer() Traces {
  return Traces{}
}

// UpdateTrace appends array to TraceInfo
func (t *Traces) UpdateTrace(req *http.Request, res *http.Response, respBody interface{}) {
  t.Trace = append(t.Trace, TraceInfo{transformRequest(req),transformResponse(res, respBody)})
}

// channels are not support in json marshal so we have to omit channel from request struct
func transformRequest(req *http.Request) TraceRequest {
  tr := TraceRequest{req.Method,
    req.URL,
    req.Proto,
    req.Header ,
    req.ContentLength,
    req.Host,
    req.Form,
    req.PostForm,
    req.MultipartForm,
    req.Trailer,
    req.RemoteAddr,
    req.RequestURI}
    return tr
}

func transformResponse(resp *http.Response, respBody interface{}) TraceResponse {
  tr := TraceResponse{
    resp.Status,
    resp.StatusCode,
    resp.Proto,
    resp.Header,
    resp.ContentLength, 
    resp.TransferEncoding, 
    resp.Trailer,
    respBody}
  return tr
}
