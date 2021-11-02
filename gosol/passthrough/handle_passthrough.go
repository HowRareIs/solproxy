package handle_passthrough

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/slawomir-pryczek/handler_socket2"
)

type Handle_passthrough struct {
}

func (this *Handle_passthrough) Initialize() {
}

func (this *Handle_passthrough) Info() string {
	return "Passthrough plugin"
}

func (this *Handle_passthrough) GetActions() []string {
	return []string{"adv"}
}

func (this *Handle_passthrough) HandleAction(action string, data *handler_socket2.HSParams) string {

	pt_url := handler_socket2.Config().Get("PASSTHROUGH_URL", "")
	if len(pt_url) == 0 {
		return "Please specify PASSTHROUGH_URL"
	}
	if strings.Index(strings.ToLower(pt_url), "http://") == -1 &&
		strings.Index(strings.ToLower(pt_url), "https://") == -1 {
		pt_url = "http://" + pt_url
	}

	ret_error := func(e error) string {
		if e == nil {
			return ""
		}
		ret := make(map[string]string)
		ret["error"] = e.Error()
		tmp, _ := json.Marshal(ret)
		return string(tmp)
	}

	req, err := http.NewRequest("GET", pt_url, nil)
	if err != nil {
		return ret_error(err)
	}

	q := req.URL.Query()
	for k, v := range data.GetParamsS() {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// run the request using client
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		TLSHandshakeTimeout: 15 * time.Second,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return ret_error(err)
	}

	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret_error(err)
	}
	data.FastReturnB(resp_body)
	return ""
}
