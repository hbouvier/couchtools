package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/logutils"
	"github.com/hbouvier/httpclient"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PutJsonResult struct {
	Ok  bool   `json:"ok"`
	Id  string `json:"id"`
	Rev string `json:"rev"`
}

func main() {
	serverPtr := flag.String("server", "http://localhost:5984", "Server address and port")
	databasePtr := flag.String("database", "", "URL to the design document")
	userPtr := flag.String("user", "", "User name")
	passwordPtr := flag.String("password", "", "User password")
	basePathPtr := flag.String("path", ".", "Path of the design documents")
	levelPtr := flag.String("log", "INFO", "Logging level")
	ignoreRev := flag.Bool("ignore-rev", false, "Ignore the _rev.js files")
	flag.Parse()
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		MinLevel: logutils.LogLevel(*levelPtr),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	var command string
	var documentId string
	switch len(flag.Args()) {
	case 2:
		command = flag.Args()[0]
		if command != "upload" && command != "download" {
			usage(flag.Args())
		}
		documentId = flag.Args()[1]
	case 0:
		usage(flag.Args())
	default:
		usage(flag.Args())
	}
	if *databasePtr == "" {
		fmt.Printf("-database=name is a required parameters\n")
		usage(flag.Args())
	}

	var basicAuthentication *httpclient.BasicAuthentication = nil
	if *userPtr != "" && *passwordPtr != "" {
		basicAuthentication = &httpclient.BasicAuthentication{Name: *userPtr, Secret: *passwordPtr}
	}
	client := httpclient.New(*serverPtr, basicAuthentication, map[string]string{})

	switch command {
	case "download":
		download(client, *databasePtr, *basePathPtr, documentId)
	case "upload":
		upload(client, *databasePtr, *basePathPtr, documentId, *ignoreRev)
	default:
		panic("Should never reach this!")
	}
}

func usage(args []string) {
	if len(args) != 0 {
		fmt.Printf("%s <- Unknown arguments\n", strings.Join(args, ", "))
	}

	fmt.Printf("USAGE: couchtools -server=http://localhost:5984 -user admin -password Secret -path=${HOME}/couchdb -database=my-database [download|upload] _design/indexes\n")
	os.Exit(2)
}

func download(client httpclient.Client, database string, path string, documentId string) {
	url := "/" + database + "/" + documentId
	var response map[string]interface{}
	if fetchError := client.Get(url, &response); fetchError != nil {
		log.Printf("[ERROR] fetching %s", url)
		os.Exit(-1)
	}
	id, idIsString := response["_id"].(string)
	if !idIsString {
		log.Printf("[ERROR] parsing design document id %s", response["_id"])
		os.Exit(-1)
	}
	designDocName, parseDocIdError := designDocumentName(id)
	if parseDocIdError != nil {
		log.Printf("[ERROR] parsing %s >>> %#v", id, parseDocIdError)
		os.Exit(-1)
	}
	log.Printf("[INFO] downloading files to: %s", path+"/"+database)
	mkdirErr := os.MkdirAll(path+"/"+database+"/"+designDocName, 0755)
	if mkdirErr != nil {
		log.Printf("[ERROR] creating %s >>> %#v", path+"/"+database+"/"+designDocName, mkdirErr)
		os.Exit(-1)
	}
	recurseDocument(response, path+"/"+database, designDocName)
}

func designDocumentName(id string) (string, error) {
	r, err := regexp.Compile("^_design/(.*)$")
	if err != nil {
		return "", err
	}
	captured := r.FindStringSubmatch(id)
	if len(captured) == 2 {
		return captured[1], nil
	} else {
		return id, nil
	}
}

func recurseDocument(document map[string]interface{}, base string, path string) {
	for key, value := range document {
		switch child := value.(type) {
		case string:
			filename := base + "/" + path + "/" + key + ".js"
			rawContent := []byte(child)
			log.Printf("[INFO] %5d bytes <- %s", len(rawContent), filename[len(base)+1:])
			writeError := ioutil.WriteFile(filename, rawContent, 0644)
			if writeError != nil {
				log.Printf("[ERROR] writing file %s >>> %#v", filename, writeError)
				os.Exit(-1)
			}

		case map[string]interface{}:
			dirname := base + "/" + path + "/" + key
			log.Printf("[DEBUG] mkdir %s", dirname)
			mkdirErr := os.MkdirAll(dirname, 0755)
			if mkdirErr != nil {
				log.Printf("[ERROR] creating %s >>> %#v", dirname, mkdirErr)
				os.Exit(-1)
			}
			recurseDocument(child, base, path+"/"+key)

		default:
			log.Printf("[ERROR] %s is trouble!", base+"/"+path+"/"+key)
		}
	}
}

func upload(client httpclient.Client, database string, path string, documentId string, ignoreRev bool) {
	designDocName, parseDocIdError := designDocumentName(documentId)
	if parseDocIdError != nil {
		log.Printf("[ERROR] parsing %s >>> %#v", documentId, parseDocIdError)
		os.Exit(-1)
	}

	desingDocuments := make(map[string]interface{})
	fullpath := path + "/" + database + "/" + designDocName
	log.Printf("[INFO] path to recurse: %s", fullpath)
	filepath.Walk(fullpath, RecursePath(fullpath, "*.js", desingDocuments))
	if ignoreRev == true {
		delete(desingDocuments, "_rev")
	}

	url := "/" + database + "/" + documentId
	response := PutJsonResult{}
	err := client.Put(url, desingDocuments, &response)
	if err != nil {
		log.Printf("[ERROR] Upload error of %s >>> %#v", url, err)
		os.Exit(-1)
	}
	log.Printf("[INFO] %#v", response)
	writeError := ioutil.WriteFile(fullpath+"/_rev.js", []byte(response.Rev), 0644)
	if writeError != nil {
		log.Printf("[ERROR] Unable to update the REVision (%s) into %s >>> %#v", response.Rev, fullpath+"/_rev.js", writeError)
		os.Exit(-1)
	}
}

func RecursePath(path string, filter string, desingDocuments map[string]interface{}) func(string, os.FileInfo, error) error {
	return func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[WARN] RecursePath >>> %#v", err)
			return nil // ignore.
		}
		if fp == path {
			log.Printf("[DEBUG] RecursePath skipping %s", fp)
			return nil // ignore.
		}

		matched, err := filepath.Match(filter, fi.Name())
		if err != nil {
			log.Printf("[ERROR] RecursePath INVALID FILTER %s >>> %#v", filter, err)
			return err // bailout!
		}
		log.Printf("[DEBUG] path: %s, fp: %s, fi: %s", path, fp, fi.Name())
		log.Printf("[DEBUG] fp: %s, fi: %s", fp[len(path):], fi.Name())
		log.Printf("[DEBUG] key: %s", removeBasePath(path, fp))
		if fi.IsDir() {
			put(removeBasePath(path, fp), make(map[string]interface{}), desingDocuments)
		} else if matched {
			log.Printf("[DEBUG] RecursePath file %s", fp)
			rawContent, readError := ioutil.ReadFile(fp)
			if readError != nil {
				log.Printf("[ERROR] RecursePath read error %s >>> %#v", fp, readError)
				return readError // bailout!
			}
			key := removeBasePath(path, fp[:len(fp)-3])
			log.Printf("[INFO] %5d bytes -> %s", len(rawContent), key)
			put(key, string(rawContent), desingDocuments)
		}
		return nil
	}
}

func removeBasePath(basePath string, path string) string {
	log.Printf("[DEBUG] basepath: %s -> path: %s", basePath, path)
	log.Printf("[DEBUG] basepath[-1]: %s", basePath[len(basePath)-1:])
	log.Printf("[DEBUG] with/:%s without/:%sbasepath[-1]: %s", path[len(basePath):], path[len(basePath)+1:])
	if basePath[len(basePath)-1:] == "/" {
		return path[len(basePath):]
	} else {
		return path[len(basePath)+1:]
	}
}

func put(key string, value interface{}, dic map[string]interface{}) {
	keys := strings.Split(key, "/")
	for _, k := range keys[:len(keys)-1] {
		switch d := dic[k].(type) {
		case map[string]interface{}:
			dic = d
		default:
			log.Printf("[ERROR] type: %s", d)
			panic("")
		}
	}
	dic[keys[len(keys)-1:][0]] = value
}
