package testreport

// original source: https://github.com/steinfletcher/apitest (MIT License)

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/steinfletcher/apitest"
)

type (
	htmlTemplateModel struct {
		Title          string
		SubTitle       string
		StatusCode     int
		BadgeClass     string
		LogEntries     []logEntry
		WebSequenceDSL string
	}

	logEntry struct {
		Header    string
		Body      string
		Timestamp time.Time
	}

	// SequenceDiagramFormatter implementation of a ReportFormatter
	SequenceDiagramFormatter struct {
		storagePath      string
		fs               fileSystem
		internalRecorder *apitest.Recorder
	}

	fileSystem interface {
		create(name string) (*os.File, error)
		mkdirAll(path string, perm os.FileMode) error
	}

	osFileSystem struct{}

	webSequenceDiagramDSL struct {
		data  bytes.Buffer
		count int
	}
)

func (r *osFileSystem) create(name string) (*os.File, error) {
	return os.Create(name)
}

func (r *osFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *webSequenceDiagramDSL) addRequestRow(source string, target string, description string) {
	r.addRow("->", source, target, description)
}

func (r *webSequenceDiagramDSL) addResponseRow(source string, target string, description string) {
	r.addRow("->>", source, target, description)
}

func (r *webSequenceDiagramDSL) addRow(operation, source string, target string, description string) {
	r.count++
	r.data.WriteString(fmt.Sprintf("%s%s%s: (%d) %s\n",
		source,
		operation,
		target,
		r.count,
		description))
}

func (r *webSequenceDiagramDSL) addParticipant(name string) {
	r.data.WriteString(fmt.Sprintf("Participant %s\n", name))
}

func (r *webSequenceDiagramDSL) toString() string {
	return r.data.String()
}

// Format formats the events received by the recorder
func (r *SequenceDiagramFormatter) Format(recorder *apitest.Recorder) {
	r.internalRecorder.AddMeta(recorder.Meta)
	r.internalRecorder.AddTitle(recorder.SubTitle)

	r.internalRecorder.Events = append(r.internalRecorder.Events, recorder.Events...)
}

func (r *SequenceDiagramFormatter) Flush() {
	output, err := newHTMLTemplateModel(r.internalRecorder)
	if err != nil {
		panic(err)
	}

	tmpl, err := htmlTemplate.New("sequenceDiagram").
		Funcs(*templateFunc).
		Parse(reportTemplate)
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, output)
	if err != nil {
		panic(err)
	}

	fileName := fmt.Sprintf("%s.html", "index")
	err = r.fs.mkdirAll(r.storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	saveFilesTo := fmt.Sprintf("%s/%s", r.storagePath, fileName)

	f, err := r.fs.create(saveFilesTo)
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString(out.String())
	if err != nil {
		panic(err)
	}
}

// SequenceDiagram produce a sequence diagram at the given path or .sequence by default
func SequenceDiagram(path ...string) *SequenceDiagramFormatter {
	var storagePath string
	if len(path) == 0 {
		storagePath = ".sequence"
	} else {
		storagePath = path[0]
	}
	return &SequenceDiagramFormatter{
		storagePath:      storagePath,
		fs:               &osFileSystem{},
		internalRecorder: apitest.NewTestRecorder(),
	}
}

var templateFunc = &htmlTemplate.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
}

func formatDiagramRequest(req *http.Request) string {
	out := req.URL.Path
	if req.URL.RawQuery != "" {
		out = fmt.Sprintf("%s %s?%s", req.Method, out, req.URL.RawQuery)
	}
	if len(out) > 65 {
		return fmt.Sprintf("%s...", out[:65])
	}
	return out
}

func badgeCSSClass(status int) string {
	class := "badge badge-success"
	if status >= 400 && status < 500 {
		class = "badge badge-warning"
	} else if status >= 500 {
		class = "badge badge-danger"
	}
	return class
}

func newHTMLTemplateModel(r *apitest.Recorder) (htmlTemplateModel, error) {
	if len(r.Events) == 0 {
		return htmlTemplateModel{}, errors.New("no events are defined")
	}
	var logs []logEntry
	webSequenceDiagram := &webSequenceDiagramDSL{}
	webSequenceDiagram.addParticipant(apitest.ConsumerDefaultName)
	webSequenceDiagram.addParticipant(apitest.SystemUnderTestDefaultName)

	for _, event := range r.Events {
		switch v := event.(type) {
		case apitest.HttpRequest:
			httpReq := v.Value
			webSequenceDiagram.addRequestRow(v.Source, v.Target, formatDiagramRequest(httpReq))
			entry, err := newHTTPRequestLogEntry(httpReq)
			if err != nil {
				return htmlTemplateModel{}, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case apitest.HttpResponse:
			webSequenceDiagram.addResponseRow(v.Source, v.Target, strconv.Itoa(v.Value.StatusCode))
			entry, err := newHTTPResponseLogEntry(v.Value)
			if err != nil {
				return htmlTemplateModel{}, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case apitest.MessageRequest:
			webSequenceDiagram.addRequestRow(v.Source, v.Target, v.Header)
			logs = append(logs, logEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		case apitest.MessageResponse:
			webSequenceDiagram.addResponseRow(v.Source, v.Target, v.Header)
			logs = append(logs, logEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		default:
			panic("received unknown event type")
		}
	}

	status, err := r.ResponseStatus()
	if err != nil {
		return htmlTemplateModel{}, err
	}

	return htmlTemplateModel{
		WebSequenceDSL: webSequenceDiagram.toString(),
		LogEntries:     logs,
		Title:          r.Title,
		StatusCode:     status,
		BadgeClass:     badgeCSSClass(status),
	}, nil
}

func newHTTPRequestLogEntry(req *http.Request) (logEntry, error) {
	reqHeader, err := httputil.DumpRequest(req, false)
	if err != nil {
		return logEntry{}, err
	}
	body, err := formatBodyContent(req.Body, func(replacementBody io.ReadCloser) {
		req.Body = replacementBody
	})
	if err != nil {
		return logEntry{}, err
	}
	return logEntry{Header: string(reqHeader), Body: body}, err
}

func newHTTPResponseLogEntry(res *http.Response) (logEntry, error) {
	resDump, err := httputil.DumpResponse(res, false)
	if err != nil {
		return logEntry{}, err
	}
	body, err := formatBodyContent(res.Body, func(replacementBody io.ReadCloser) {
		res.Body = replacementBody
	})
	if err != nil {
		return logEntry{}, err
	}
	return logEntry{Header: string(resDump), Body: body}, err
}

func formatBodyContent(bodyReadCloser io.ReadCloser, replaceBody func(replacementBody io.ReadCloser)) (string, error) {
	if bodyReadCloser == nil {
		return "", nil
	}

	body, err := ioutil.ReadAll(bodyReadCloser)
	if err != nil {
		return "", err
	}

	replaceBody(ioutil.NopCloser(bytes.NewReader(body)))

	buf := new(bytes.Buffer)
	if json.Valid(body) {
		jsonEncodeErr := json.Indent(buf, body, "", "    ")
		if jsonEncodeErr != nil {
			return "", jsonEncodeErr
		}
		s := buf.String()
		return s, nil
	}

	_, err = buf.Write(body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
