// api_calls.go

package maz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/git719/utl"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type jsonT map[string]interface{} // Local syntactic sugar, for easier reading
type strMapT map[string]string

func ApiGet(url string, z Bundle, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("GET", url, z, nil, params, false) // false = quiet, for normal ops
}

func ApiGetDebug(url string, z Bundle, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("GET", url, z, nil, params, true) // true = verbose, for debugging
}

func ApiPost(url string, z Bundle, payload jsonT, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("POST", url, z, payload, params, false) // false = quiet, for normal ops
}

func ApiPostDebug(url string, z Bundle, payload jsonT, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("POST", url, z, payload, params, true) // true = verbose, for debugging
}

func ApiPut(url string, z Bundle, payload jsonT, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("PUT", url, z, payload, params, false) // false = quiet, for normal ops
}

func ApiPutDebug(url string, z Bundle, payload jsonT, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("PUT", url, z, payload, params, true) // true = verbose, for debugging
}

func ApiDelete(url string, z Bundle, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("DELETE", url, z, nil, params, false) // false = quiet, for normal ops
}

func ApiDeleteDebug(url string, z Bundle, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("DELETE", url, z, nil, params, true) // true = verbose, for debugging
}

func ApiCall(method, url string, z Bundle, payload jsonT, params strMapT, verbose bool) (result jsonT, rsc int, err error) {
	// Make API call and return JSON object, Response StatusCode, and error. See https://eager.io/blog/go-and-json/
	// for a clear explanation of how to interpret JSON responses with GoLang

	if !strings.HasPrefix(url, "http") {
		utl.Die(utl.Trace() + "Error: Bad URL, " + url + "\n")
	}

	// Map headers to corresponding API endpoint
	var headers strMapT = nil
	if strings.HasPrefix(url, ConstMgUrl) {
		headers = z.MgHeaders
	} else if strings.HasPrefix(url, ConstMgUrl) {
		headers = z.AzHeaders
	}

	// Set up new HTTP request client
	client := &http.Client{Timeout: time.Second * 60} // One minute timeout
	var req *http.Request = nil
	switch strings.ToUpper(method) {
	case "GET":
		req, err = http.NewRequest("GET", url, nil)
	case "POST":
		jsonData, err := json.Marshal(payload)
		if err != nil {
			panic(err.Error())
		}
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	case "PUT":
		jsonData, err := json.Marshal(payload)
		if err != nil {
			panic(err.Error())
		}
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	case "DELETE":
		req, err = http.NewRequest("DELETE", url, nil)
	default:
		utl.Die(utl.Trace() + "Error: Unsupported HTTP method\n")
	}
	if err != nil {
		panic(err.Error())
	}

	// Set up the headers
	for h, v := range headers {
		req.Header.Add(h, v)
	}

	// Set up the query parameters and encode
	reqParams := req.URL.Query()
	for p, v := range params {
		reqParams.Add(p, v)
	}
	req.URL.RawQuery = reqParams.Encode()

	// === MAKE THE CALL ============
	if verbose {
		fmt.Println(utl.Blu("==== REQUEST ================================="))
		fmt.Println(method + " " + url)
		PrintHeaders(req.Header)
		PrintParams(reqParams)
		if payload != nil {
			fmt.Println(utl.Blu("payload") + ":")
			utl.PrintJsonColor(payload)
		}
	}
	r, err := client.Do(req) // Make the call
	if err != nil {
		panic(err.Error())
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body) // Read the response body
	if err != nil {
		panic(err.Error())
	}
	// This function caters to Microsoft Azure REST API calls. Note that variable 'body' is of type
	// []uint8, which is essentially a long string that evidently can be either: 1) a single integer
	// number, or 2) a JSON object string that needs unmarshalling. Below conditional is based on
	// this interpretation, but may need confirmation then better handling

	// Create jsonResult variable object to be return
	var jsonResult jsonT = nil
	if intValue, err := strconv.ParseInt(string(body), 10, 64); err == nil {
		// It's an integer, probably an API object count value
		jsonResult = make(map[string]interface{})
		jsonResult["value"] = intValue
	} else {
		// It's a regular JSON result, or null
		if len(body) > 0 { // Make sure we have something to unmarshal, else guaranteed panic
			if err = json.Unmarshal([]byte(body), &jsonResult); err != nil {
				panic(err.Error())
			}
		}
		// If it's null, returning r.StatusCode below will let caller know
	}
	if verbose {
		fmt.Println(utl.Blu("==== RESPONSE ================================"))
		fmt.Printf("%s: %d %s\n", utl.Blu("status"), r.StatusCode, http.StatusText(r.StatusCode))
		fmt.Println(utl.Blu("result") + ":")
		utl.PrintJsonColor(jsonResult)
		resHeaders, err := httputil.DumpResponse(r, false)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(utl.Blu("headers") + ":")
		fmt.Println(string(resHeaders))
	}
	return jsonResult, r.StatusCode, err
}

func ApiErrorCheck(method, url, caller string, r jsonT) {
	// Print useful error information
	if r["error"] != nil {
		e := r["error"].(map[string]interface{})
		errMsg := method + " " + url + "\n" + caller + "Error: " + e["message"].(string) + "\n"
		fmt.Printf(utl.Red(errMsg))
	}
}

func PrintHeaders(headers http.Header) {
	// HTTP headers printing fuction specific to API calls. Simplifies ApiCall function
	if headers == nil {
		return
	}
	fmt.Println(utl.Blu("headers") + ":")
	for k, v := range headers {
		fmt.Printf("  %s:\n", utl.Blu(k))
		count := len(v) // Array of string
		if count == 1 {
			fmt.Printf("    - %s\n", utl.Gre(string(v[0]))) // In YAML-like output, 1st entry gets the dash
		}
		if count > 2 {
			for _, i := range v[1:] {
				fmt.Printf("      %s\n", utl.Gre(string(i)))
			}
		}
	}
}

func PrintParams(params url.Values) {
	// HTTP parameters printing fuction specific to API calls. Simplifies ApiCall function
	if params == nil {
		return
	}
	fmt.Println(utl.Blu("params") + ":")
	for k, v := range params {
		fmt.Printf("  %s:\n", utl.Blu(k))
		count := len(v) // Array of string
		if count == 1 {
			fmt.Printf("    - %s\n", utl.Gre(string(v[0]))) // In YAML-like output, 1st entry gets the dash
		}
		if count > 2 {
			for _, i := range v[1:] {
				fmt.Printf("      %s\n", utl.Gre(string(i)))
			}
		}
	}
}
