package resttemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
)

type RestTemplate interface {
	Post(url string, body interface{}) []byte
	PostWithHeaders(url string, body interface{}, headers map[string]string) []byte
	Get(url string) []byte
	GetWithAuthorization(url string, header map[string]string) []byte
	PostMultipart(url string, body *bytes.Buffer, writer *multipart.Writer) []byte
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

type ContentFile struct {
	FileName  string
	File      []byte
	FieldName string
}

func (r RestTemplateStruct) PostMultipart(url string, body *bytes.Buffer, writer *multipart.Writer) []byte {
	logrus.Infoln(url)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}

	// Set the content type to multipart; this is crucial for file uploads
	req.Header.Set("X-AI_TOKEN", "AI-674ceb4b6fd801527167dd2f")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-REQUEST_FROM", "AI_TOKEN")

	// Perform the request
	client := &http.Client{}
	respRequest, err := client.Do(req)
	if err != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", err)))
	}
	defer respRequest.Body.Close()

	// Check the response status code
	if respRequest.StatusCode != http.StatusOK {
		panic(exception.NewBadRequestError(fmt.Sprintf("An Error Occured with status code: %v", respRequest.StatusCode)))
	}

	// Read the response body
	bodyResp, errParse := ioutil.ReadAll(respRequest.Body)
	if errParse != nil {
		panic(exception.NewInternalServerError(fmt.Sprintf("An Error Occured %v", errParse)))
	}
	logrus.Infoln(string(bodyResp))

	return bodyResp
}

func NewRestTemplate() RestTemplate {
	return &RestTemplateStruct{}
}
