package resttemplate

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"path/filepath"
	"testing"
)

func TestUploadFile(t *testing.T) {
	data := map[string]interface{}{
		"name": "Muhammad Suryono",
	}

	df, _ := json.Marshal(data)
	r := NewRestTemplate()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f := ContentFile{
		FileName:  "f.json",
		File:      df,
		FieldName: "pdf_file",
	}
	part, err := w.CreateFormFile(f.FieldName, filepath.Base(f.FileName))
	if err != nil {
		return
	}

	reader := bytes.NewReader(df)
	_, err = io.Copy(part, reader)
	if err != nil {
		return
	}
	// Don't forget to close the writer
	if err := w.Close(); err != nil {
		return
	}
	r.PostMultipart("https://ai-dashboard-api.primeskills.space/gateway/ai/conversational/file-upload?index_name=EJOurney", &b, w)
}
