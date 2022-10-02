package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type LastError struct {
	str     string
	details string

	call    string
	call_ts int64
	counter int
}

func isHTTPError(resp *http.Response, err error, post []byte) *LastError {
	if resp != nil && resp.StatusCode == 200 && err == nil {
		return nil
	}

	this := LastError{counter: -1}
	this.call = string(post)
	this.call_ts = time.Now().UnixNano() / 1000

	tmp := make([]string, 0, 10)
	if resp == nil {
		tmp = append(tmp, "No response from host")
	}
	if err != nil {
		tmp = append(tmp, "Error: "+err.Error())
	}
	if resp.StatusCode != 200 {
		tmp = append(tmp, "HTTP: "+resp.Status)
	}
	this.str = strings.Join(tmp, "\n")

	tmp = tmp[:0]
	if len(resp.Header) > 0 {
		tmp = append(tmp, "Response Headers:")
		for k, v := range resp.Header {
			tmp = append(tmp, k+": "+strings.Join(v, ", "))
		}
		tmp = append(tmp, "")
	}

	body := []byte(nil)
	if resp.Body != nil {
		body, _ = ioutil.ReadAll(resp.Body)
	}
	if body != nil {
		tmp = append(tmp, "Body: "+string(body))
	} else {
		tmp = append(tmp, "Body: -")
	}

	this.details = strings.Join(tmp, "\n")
	return &this
}

func isGenericError(err error, post []byte) *LastError {
	if err == nil {
		return nil
	}

	this := LastError{counter: -1}
	this.call = string(post)
	this.call_ts = time.Now().UnixNano() / 1000

	this.str = err.Error()
	this.details = "-"
	return &this
}

func (this LastError) String() string {
	tmp := fmt.Sprintf("Error %d @%s", this.counter, time.UnixMicro(this.call_ts).Format("2006-01-02 15:04:05")) + " / " + this.str + "\n"
	tmp += "Request Data:" + this.call + "\n\n"
	tmp += this.details
	return tmp
}
