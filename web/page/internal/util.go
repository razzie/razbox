package internal

import (
	"fmt"
	"io/ioutil"
)

// GetContentTemplate returns the content template for a page
func GetContentTemplate(page string) string {
	t, err := ioutil.ReadFile(fmt.Sprintf("web/template/%s.template", page))
	if err != nil {
		panic(err)
	}
	return string(t)
}
