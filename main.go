package main

import (
	"bytes"
	"fmt"
	bf "github.com/russross/blackfriday/v2"
	git "gopkg.in/src-d/go-git.v4"
	"io/ioutil"
	"os"
	"text/template"
	"time"
)

const (
	HTML_EXTENSION = ".html"

	// todo(introduce flags)
	REPO          = "https://github.com/octetz/sample-md"
	TEMPLATE_FILE = "template.html"
	INPUT         = "./input"
	OUTPUT        = "./output"
)

func main() {
	in := make(chan int)
	go sync(in)
	<-in
}

// sync runs a continuous loop every 30 seconds which attemps to sync the git repo
// and parse any new md (or changes to md)
func sync(in chan int) {
	fmt.Println("Starting git/site sync")
	for {
		time.Sleep(30 * time.Second)

		err := getContent(INPUT)
		if err != nil {
			fmt.Println(err.Error())
			close(in)
		}

		err = parseDir(INPUT, OUTPUT)
		if err != nil {
			fmt.Println(err.Error())
			close(in)
		}

	}
	close(in)
}

func parseDir(input, output string) error {
	// check input if exists, return err if not
	files, err := ioutil.ReadDir(input)
	if err != nil {
		return err
	}

	// check output, if not exist, create it
	if _, err := os.Stat(output); os.IsNotExist(err) {
		// ModePerm = 0777
		os.Mkdir(output, os.ModePerm)
	}

	// for over input dir contents
	for _, f := range files {
		b, err := ioutil.ReadFile(input + string(os.PathSeparator) + f.Name())
		if err != nil {
			continue
		}

		fmt.Printf("Creating HTML for %s: ", f.Name())
		o := bf.Run(b)
		outputFile := output + string(os.PathSeparator) + f.Name() + HTML_EXTENSION
		out, err := parseTemplate(TEMPLATE_FILE, o)
		if err != nil {
			fmt.Printf("failed to parse template %s. err: %s. continuing", outputFile, err.Error())
		}
		err = ioutil.WriteFile(outputFile, out, os.ModePerm)
		if err != nil {
			fmt.Printf("failed to write %s. err: %s. continuing", outputFile, err.Error())
		}
	}

	// output html into output dir
	return nil
}

func parseTemplate(t string, html []byte) ([]byte, error) {
	rawTemp, err := ioutil.ReadFile(t)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("template.html").Parse(string(rawTemp))

	var tpl bytes.Buffer

	tmpl.Execute(&tpl, string(html))

	return tpl.Bytes(), nil
}

func getContent(input string) error {
	// check input if exists return err
	_, err := os.Open(input)
	if err != nil {
		// Clone the given repository to the given directory
		_, err = git.PlainClone(input, false, &git.CloneOptions{
			URL:      REPO,
			Progress: os.Stdout,
		})
		if err != nil {
			return err
		}
	}

	r, err := git.PlainOpen(input)
	if err != nil {
		return err
	}

	// Get the working directory for the repository
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	// Pull the latest changes from the origin remote and merge into the current branch
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		fmt.Println("Git repo already up to date")
	}

	// Print the latest commit that was just pulled
	ref, err := r.Head()
	if err != nil {
		return err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	fmt.Println(commit)

	return nil
}

/*

1. Clone runs
2. Persists data locally
     ^ is the ./input dir
3. call parseDir
4. get html
5. persist html to dir
                   ^  -- where nginx serves

*/
