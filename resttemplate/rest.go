package resttemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"io/ioutil"
	"net/http"
	"strings"
)

type RestTemplate interface {
	Post(url string, body interface{}) []byte
	PostWithHeaders(url string, body interface{}, headers map[string]string) []byte
	Get(url string) []byte
	GetWithAuthorization(url string, header map[string]string) []byte
}

type RestTemplateStruct struct {
}

func (r RestTemplateStruct) PostWithHeaders(url string, body interface{}, headers map[string]string) []byte {
	method := "POST"

	by, _ := json.Marshal(body)
	payload := strings.NewReader(string(by))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	for i, header := range headers {
		req.Header.Add(i, header)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	defer res.Body.Close()

	bodyResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	return bodyResponse
}

func (r RestTemplateStruct) GetWithAuthorization(url string, header map[string]string) []byte {
	client := &http.Client{}
	respRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	for k, s := range header {
		respRequest.Header.Set(k, s)
	}

	res, _ := client.Do(respRequest)
	defer res.Body.Close()
	bodyResp, errParse := ioutil.ReadAll(res.Body)
	if errParse != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	return bodyResp
}

func (r RestTemplateStruct) Post(url string, body interface{}) []byte {
	bodyByte, _ := json.Marshal(body)
	payload := bytes.NewBuffer(bodyByte)
	respRequest, err := http.Post(url, "application/json", payload)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	defer respRequest.Body.Close()
	bodyResp, errParse := ioutil.ReadAll(respRequest.Body)
	if errParse != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	return bodyResp
}

func (r RestTemplateStruct) Get(url string) []byte {
	respRequest, err := http.Get(url)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	defer respRequest.Body.Close()
	bodyResp, errParse := ioutil.ReadAll(respRequest.Body)
	if errParse != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	return bodyResp
}

func NewRestTemplate() RestTemplate {
	return &RestTemplateStruct{}
}
