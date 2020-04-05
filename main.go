package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/jessevdk/go-flags"
	"github.com/spf13/viper"
)

var usr = func() *user.User {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr
}()

// Option represents application options
type Option struct {
	Number bool `short:"n" long:"number" description:"Show contents with line numbers"`
}

// Config represents the settings for this application
type Config struct {
	Theme string `json:"theme"`
}

// CLI represents this application itself
type CLI struct {
	Config Config
}

func download(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Cat formats file with syntax highlighting
func (c CLI) Cat(opt Option, path string) (string, error) {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get(c.Config.Theme)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	r, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	iterator, err := lexer.Tokenise(nil, string(r))
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)
	if err := formatter.Format(w, style, iterator); err != nil {
		return "", err
	}

	s := w.String()
	ss := strings.Split(s, "\n")

	contents := ""
	for i, s := range ss {
		if opt.Number {
			contents += fmt.Sprintf("%6d  ", i+1)
		}

		contents += fmt.Sprintf("%s\n", s)
	}

	return contents, nil
}

func run(args []string) int {
	var opt Option
	args, err := flags.ParseArgs(&opt, args)
	if err != nil {
		return 2
	}

	p := path.Join(usr.HomeDir, ".config/ccat.json")

	if _, err = os.Stat(p); os.IsNotExist(err) {
		url := "https://raw.githubusercontent.com/skmatz/ccat/master/ccat.json"
		if err := download(url, p); err != nil {
			fmt.Println(err)
			return 1
		}
	}

	if len(args) == 0 {
		return 2
	}

	viper.SetConfigName("ccat")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.config")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
		return 1
	}

	var cli CLI
	if err := viper.Unmarshal(&cli); err != nil {
		fmt.Println(err)
		return 1
	}

	for _, arg := range args {
		contents, err := cli.Cat(opt, arg)
		if err != nil {
			return 1
		}
		fmt.Println(contents)
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
