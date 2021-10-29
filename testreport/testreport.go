package testreport

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/steinfletcher/apitest"
)

type TestSuiteReporter struct {
	InternalFormatter map[string]*SequenceDiagramFormatter
	Path              string
	Name              string
}

func New(name string, path string) *TestSuiteReporter {
	return &TestSuiteReporter{
		Name:              name,
		Path:              path,
		InternalFormatter: map[string]*SequenceDiagramFormatter{},
	}
}

func (r *TestSuiteReporter) Format(recorder *apitest.Recorder) {
	if _, exist := r.InternalFormatter[recorder.SubTitle]; !exist {
		path := fmt.Sprintf("%s/%s/%s", r.Path, r.Name, recorder.SubTitle)
		r.InternalFormatter[recorder.SubTitle] = SequenceDiagram(path)
	}

	formatter := r.InternalFormatter[recorder.SubTitle]
	formatter.Format(recorder)
}

const indexTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.1.2/css/bootstrap.min.css">
    <script src="https://stackpath.bootstrapcdn.com/bootstrap/4.1.2/js/bootstrap.min.js"></script>
    <style>
        body {
            padding-top: 2rem;
            padding-bottom: 2rem;
        }
    </style>
</head>
<body>
<div class="container-fluid">
	<h1>{{ .Name }}</h1>
    <table class="table">
        <thead>
		<tr>
			<th>Test</th>
			<th>Report</th>
		</tr>
        </thead>
        <tbody>
		{{ range $name, $formatter := .InternalFormatter }}
		<tr>
			<td>{{ $name }}</td>
				<td><a href="{{ $name }}/index.html">{{ $name }}</a></td>
		</tr>
		{{ end }}
        </tbody>
	</table>
</div>
</body>
</html>
`

func (r *TestSuiteReporter) Flush() {
	for _, formatter := range r.InternalFormatter {
		formatter.Flush()
	}

	var buf bytes.Buffer
	t := template.Must(template.New("index").Parse(indexTemplate))
	err := t.Execute(&buf, r)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(fmt.Sprintf("%s/%s/index.html", r.Path, r.Name))
	if err != nil {
		panic(err)
	}

	defer f.Close()
	_, err = f.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
}
