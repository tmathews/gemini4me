package main

import (
	"flag"
	"log"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"git.sr.ht/~yotam/go-gemini"
	"github.com/BurntSushi/toml"
)

func init() {
	mime.AddExtensionType(".gem", "text/gemini")
	mime.AddExtensionType(".gemini", "text/gemini")
	mime.AddExtensionType(".gmi", "text/gemini")
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "config", "./conf.toml", "The configuration file to load.")
	flag.Parse()

	c, err := LoadConfig(confPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	server := GeminiServer{
		RootDirectory: c.RootDirectory,
		DefaultFile:   c.DefaultFile,
		Extension:     c.Extension,
	}
	log.Printf("Serving... %s\n", c.Address)
	err = gemini.ListenAndServe(c.Address, c.CertificateFile, c.KeyFile, server)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("Exiting...")
}

type Config struct {
	Address         string
	CertificateFile string
	KeyFile         string
	RootDirectory   string
	DefaultFile     string
	Extension       string
}

func NewConfig() Config {
	return Config{
		Address:         "127.0.0.1:1965",
		CertificateFile: "server.cert",
		KeyFile:         "server.key",
		RootDirectory:   ".",
		DefaultFile:     "index",
		Extension:       ".gem",
	}
}

func LoadConfig(filename string) (Config, error) {
	c := NewConfig()
	_, err := toml.DecodeFile(filename, &c)
	return c, err
}

type GeminiServer struct {
	// The root directory to serve files from.
	RootDirectory string

	// This is the default file to serve for root folder if it exists.
	DefaultFile string

	// The default gem extension to look for when magic serving gemini files (/my-gem-log -> /my-gem-log.gem)
	Extension string
}

func (g GeminiServer) Handle(r gemini.Request) gemini.Response {
	return g.HandleWithLog(r)
}

func (g GeminiServer) HandleWithLog(r gemini.Request) gemini.Response {
	res := g._handle(r)
	log.Printf("(%d) %s %s\n", res.Status, r.URL, res.Meta)
	return res
}

func (g GeminiServer) _handle(r gemini.Request) gemini.Response {
	u, err := url.Parse(r.URL)
	if err != nil {
		log.Println(err.Error())
		return gemini.Response{
			Status: gemini.StatusBadRequest,
			Meta:   "Bad URL",
		}
	}

	fp := path.Join(g.RootDirectory, u.Path)
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			if filepath.Ext(fp) == "" {
				return serve(fp + g.Extension)
			}
			return notFound()
		} else {
			log.Println(err.Error())
		}
		return gemini.Response{
			Status: gemini.StatusTemporaryFailure,
			Meta:   "File Lookup Issue",
		}
	}
	if info.IsDir() {
		ifp := path.Join(fp, g.DefaultFile+g.Extension)
		info, err := os.Stat(ifp)
		if os.IsNotExist(err) {
			return notFound()
		} else if err != nil {
			log.Println(err.Error())
			return failure("File Read Issue")
		} else if info.IsDir() {
			return failure("Folder Loop")
		}
		return serve(ifp)
	}
	return serve(fp)
}

func serve(filename string) gemini.Response {
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		return notFound()
	} else if err != nil {
		log.Println(err.Error())
		return failure("Open File Issue")
	}
	return gemini.Response{
		Status: gemini.StatusSuccess,
		Meta:   mime.TypeByExtension(filepath.Ext(filename)),
		Body:   f,
	}
}

func failure(str string) gemini.Response {
	return gemini.Response{
		Status: gemini.StatusTemporaryFailure,
		Meta:   str,
	}
}

func notFound() gemini.Response {
	return gemini.Response{
		Status: gemini.StatusNotFound,
		Meta:   "Not Found!",
	}
}
