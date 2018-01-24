package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/microcosm-cc/bluemonday"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type HenryFile struct {
	Name        string
	Path        string
	SubPath     string
	Type        HenryFileType
	Data        []byte
	Body        string
	HasMetadata bool
	Metadata    *HenryFileMetadata
	Date        time.Time
}

type HenryFileMetadata struct {
	Title   string    `toml:"title"`
	Date    time.Time `toml:"date"`
	Draft   bool      `toml:"draft"`
	Summary string    `toml:"summary"`
}

type HenryDocument struct {
	Title             string
	Content           string
	ContentRaw        string
	ContentParagraphs []string
	Date              time.Time
	Draft             bool
	Summary           string
	SummaryRaw        string
}

type HenryFileType int

const (
	HenryFileTypeUnknown HenryFileType = iota
	HenryFileTypeMarkdown
)

func analyzeHenryFile(file *HenryFile, rootPath *string) error {
	err := classifyHenryFile(file, rootPath)
	if err != nil {
		return err
	}

	if file.Type == HenryFileTypeMarkdown {
		readErr := readHenryFileData(file)
		if readErr != nil {
			return readErr
		}

		metaErr := readHenryFileMetadata(file)
		if metaErr != nil {
			return metaErr
		}
	}

	info, err := os.Stat(file.Path)
	if err != nil {
		return err
	}
	file.Date = info.ModTime()

	return nil
}

func classifyHenryFile(file *HenryFile, rootPath *string) error {
	if strings.HasSuffix(file.Path, ".md") {
		file.Type = HenryFileTypeMarkdown
	} else {
		file.Type = HenryFileTypeUnknown
	}

	var tmp string
	tmp = strings.TrimPrefix(file.Path, *rootPath)
	tmp = strings.TrimSuffix(tmp, file.Name)
	file.SubPath = tmp

	return nil
}

func createHenryDocument(file *HenryFile) (*HenryDocument, error) {
	doc := &HenryDocument{}

	u := blackfriday.Run([]byte(file.Body))
	h := string(bluemonday.UGCPolicy().SanitizeBytes(u))
	content := strings.Replace(h, "\n\n", "\n", -1)
	content = strings.Trim(content, "\n")

	doc.Content = content
	doc.ContentRaw = file.Body
	doc.ContentParagraphs = make([]string, 0)
	for _, paragraph := range strings.SplitAfter(doc.Content, "\n") {
		doc.ContentParagraphs = append(doc.ContentParagraphs, strings.Trim(paragraph, "\n"))
	}

	if file.Metadata.Title != "" {
		doc.Title = file.Metadata.Title
	} else {
		doc.Title = file.Name
	}

	if !file.Metadata.Date.IsZero() {
		doc.Date = file.Metadata.Date
	} else {
		doc.Date = file.Date
	}

	if file.Metadata.Draft {
		doc.Draft = file.Metadata.Draft
	} else {
		doc.Draft = false
	}

	if file.Metadata.Summary != "" {
		su := blackfriday.Run([]byte(file.Metadata.Summary))
		sh := string(bluemonday.UGCPolicy().SanitizeBytes(su))
		doc.Summary = sh
		doc.SummaryRaw = file.Metadata.Summary
	} else {
		if len(doc.ContentParagraphs) > 0 {
			doc.Summary = doc.ContentParagraphs[0]
		}
	}

	return doc, nil
}

func createHenryDocuments(files []*HenryFile) ([]*HenryDocument, error) {
	docs := make([]*HenryDocument, 0)

	for _, file := range files {
		doc, err := createHenryDocument(file)
		if err != nil {
			debug("%s", err.Error())
			continue
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

func debug(params ...string) {
	args := make([]interface{}, len(params)-1)
	for i := 1; i < len(params); i++ {
		args[i-1] = params[i]
	}

	fmt.Printf(fmt.Sprintf("%s\n", params[0]), args...)
}

func findHenryFiles(rootPath string) ([]*HenryFile, error) {
	foundFiles := make([]*HenryFile, 0)

	err := filepath.Walk(rootPath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !file.IsDir() {
			henryFile := &HenryFile{Name: file.Name(), Path: path}
			if err := analyzeHenryFile(henryFile, &rootPath); err == nil {
				foundFiles = append(foundFiles, henryFile)
			} else {
				return err
			}
		}
		return nil
	})

	return foundFiles, err
}

func main() {
	fmt.Printf("%s v.0.1\n", os.Args[0])

	rootPath := "/Users/claes/go/src/github.com/claesp/henry/data/"
	henryFiles, err := findHenryFiles(rootPath)
	if err != nil {
		panic(err)
	}

	henryDocs, err := createHenryDocuments(henryFiles)
	if err != nil {
		panic(err)
	}

	for _, henryDoc := range henryDocs {
		fmt.Println(henryDoc)
	}
}

func readHenryFileData(file *HenryFile) error {
	fo, err := os.Open(file.Path)
	if err != nil {
		return err
	}
	defer fo.Close()

	data, err := ioutil.ReadAll(fo)
	if err != nil {
		return err
	}

	file.Data = data

	return nil
}

func readHenryFileMetadata(file *HenryFile) error {
	var metadata HenryFileMetadata

	if len(file.Data) == 0 {
		return nil
	}

	file.HasMetadata = false
	file.Metadata = &metadata

	hdr := string(file.Data[0:3])
	if hdr != "---" {
		file.Body = string(file.Data)
		return nil
	}

	headerParts := strings.Split(string(file.Data), "---")
	if _, err := toml.Decode(headerParts[1], &metadata); err != nil {
		return errors.New(fmt.Sprintf("error parsing metadata in '%s': %s", file.Name, err))
	}

	file.HasMetadata = true
	file.Metadata = &metadata
	file.Body = headerParts[2]

	return nil
}
