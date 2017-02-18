import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"fmt"
)

func FakeResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
	}
}

func DumpHttpRequestUnsafe(req *http.Request, body bool) string {
	reqBytes, _ := httputil.DumpRequest(req, body)
	return fmt.Sprintf("%s", reqBytes)
}
