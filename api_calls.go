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
	"strconv"
	"strings"
	"time"
)

type jsonT map[string]interface{} // Local syntactic sugar, for easier reading
type strMapT map[string]string

func ApiGet(url string, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("GET", url, nil, headers, params, false) // false = quiet, for normal ops
}

func ApiGetDebug(url string, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("GET", url, nil, headers, params, true) // true = verbose, for debugging
}

func ApiDelete(url string, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("DELETE", url, nil, headers, params, false) // false = quiet, for normal ops
}

func ApiDeleteDebug(url string, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("DELETE", url, nil, headers, params, true) // true = verbose, for debugging
}

func ApiPut(url string, payload jsonT, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("PUT", url, payload, headers, params, false) // false = quiet, for normal ops
}

func ApiPutDebug(url string, payload jsonT, headers, params strMapT) (result jsonT, rsc int, err error) {
	return ApiCall("PUT", url, payload, headers, params, true) // true = verbose, for debugging
}

func ApiCall(method, url string, payload jsonT, headers, params strMapT, verbose bool) (result jsonT, rsc int, err error) {
	// Make API call and return JSON object, Response StatusCode, and error. See https://eager.io/blog/go-and-json/
	// for a clear explanation of how to interpret JSON responses with GoLang

	if !strings.HasPrefix(url, "http") {
		utl.Die(utl.Trace() + "Error: Bad URL, " + url + "\n")
	}

	// Set up new HTTP client
	client := &http.Client{Timeout: time.Second * 60} // One minute timeout
	var req *http.Request = nil
	switch strings.ToUpper(method) {
	case "GET":
		req, err = http.NewRequest("GET", url, nil)
	case "POST":
		jsonData, ok := json.Marshal(payload)
		if ok != nil {
			panic(err.Error())
		}
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	case "PUT":
		jsonData, ok := json.Marshal(payload)
		if ok != nil {
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
	q := req.URL.Query()
	for p, v := range params {
		q.Add(p, v)
	}
	req.URL.RawQuery = q.Encode()

	// === MAKE THE CALL ============
	if verbose {
		fmt.Printf(utl.Cya("==== REQUEST =================================") + "\n")
		fmt.Printf(method + " " + url + "\n")
		fmt.Printf("HEADERS:\n")
		utl.PrintJson(req.Header)
		fmt.Println()
		fmt.Println("PARAMS:")
		utl.PrintJson(q)
		fmt.Println()
		// fmt.Println("PAYLOAD:")
		// utl.PrintJson(jsonData)
		// fmt.Println()
	}
	r, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}

	if verbose {
		fmt.Printf(utl.Cya("==== RESPONSE ================================") + "\n")
		fmt.Printf("%s %d %s\n", utl.Cya("STATUS:"), r.StatusCode, http.StatusText(r.StatusCode))
		fmt.Printf(utl.Cya("RESULT:") + "\n")
		utl.PrintJson(body)
		fmt.Printf("\n")
		resHeaders, err := httputil.DumpResponse(r, false)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("%s\n%s\n", utl.Cya("HEADERS:"), string(resHeaders))
	}

	// This function caters to Microsoft Azure REST API calls. Note that variable 'body' is of type
	// []uint8, which is essentially a long string that evidently can be either: 1) a single integer
	// number, or 2) a JSON object string that needs unmarshalling. Below conditional is based on
	// this interpretation, but may need confirmation then better handling

	// Create jsonResult variable object to be return
	var jsonResult jsonT = nil
	if intValue, err := strconv.ParseInt(string(body), 10, 64); err == nil {
		// It's an integer an API object count value
		jsonResult = make(map[string]interface{})
		jsonResult["value"] = intValue
	} else {
		// It's a regular JSON
		if err = json.Unmarshal([]byte(body), &jsonResult); err != nil {
			panic(err.Error())
		}
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
