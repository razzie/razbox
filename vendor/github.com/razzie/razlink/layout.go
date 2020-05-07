package razlink

import (
	"html/template"
	"net/http"
)

var layoutT = `
<!DOCTYPE html>
<html>
	<head>
		{{if .Title}}<title>{{.Title}}</title>{{end}}
		<base href="{{.Base}}" />
		<link rel="icon" href="favicon.png" type="image/png" />
		<meta name="author" content="Gábor Görzsöny" />
		<style>
		body {
			background-color: white;
			background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='4' height='4' viewBox='0 0 4 4'%3E%3Cpath fill='%23808080' fill-opacity='0.5' d='M1 3h1v1H1V3zm2-2h1v1H3V1z'%3E%3C/path%3E%3C/svg%3E");
		}
		div.outer {
			display: flex;
			align-items: center;
			justify-content: center;
		}
		div.inner {
			background-color: white;
			border: 1px solid black;
			padding: 1rem;
			display: inline-flex;
		}
		input {
			border: 0;
			outline: 0;
			background: transparent;
			border-bottom: 1px solid black;
			margin-bottom: 1rem;
		}
		table {
			border-collapse: collapse;
			margin-bottom: 1rem;
		}
		tr:nth-child(odd) > td {
			background-color: whitesmoke;
		}
		tr:nth-child(1) > td {
			font-weight: bold;
			border-bottom: 1px solid black;
			background-color: white;
		}
		td {
			padding: 10px;
		}
		</style>
	</head>
	<body>
		<div class="outer">
			<div class="inner">
				<div>
					{{template "page" .Data}}
				</div>
			</div>
		</div>
	</body>
</html>
`

var layout = template.Must(template.New("layout").Parse(layoutT))

// LayoutRenderer is a function that renders a html page
type LayoutRenderer func(w http.ResponseWriter, r *http.Request, title string, data interface{})

// BindLayout creates a layout renderer function
func BindLayout(pageTemplate string) (LayoutRenderer, error) {
	cloneLayout, _ := layout.Clone()
	tmpl, err := cloneLayout.New("page").Parse(pageTemplate)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request, title string, data interface{}) {
		var view struct {
			Title string
			Base  string
			Data  interface{}
		}

		view.Title = title
		view.Base = GetBase(r)
		view.Data = data

		tmpl.ExecuteTemplate(w, "layout", &view)
	}, nil
}
