package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// ShowFile - display html file
func ShowFile(w http.ResponseWriter, fileName string) {
	var htmlString string
	htmlBytes, err := ioutil.ReadFile("files/" + fileName)
	if err != nil {
		htmlString = "File Not Found : " + fileName
	} else {
		htmlString = string(htmlBytes)
	}
	fmt.Fprintf(w, htmlString)
}
