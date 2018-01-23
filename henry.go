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
)

type HenryFile struct {
	Name        string
	Path        string
	SubPath     string
	Type        HenryFileType
	Data        []byte
	HasMetadata bool
	Metadata    *HenryFileMetadata
	Date        time.Time
}

type HenryFileMetadata struct {
	Title string `toml:"title"`
	Date  string `toml:"date"`
	Draft bool   `toml:"draft"`
}

type HenryDocument struct {
	Title             string
	Content           string
	ContentRaw        string
	ContentParagraphs []string
	Date              time.Time
	Draft             bool
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

func createHenryDocuments(files []*HenryFile) ([]*HenryDocument, error) {
	docs := make([]*HenryDocument, 0)

	return docs, nil
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

	for _, henryFile := range henryFiles {
		fmt.Println(henryFile)
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
	if len(file.Data) == 0 {
		return nil
	}

	hdr := string(file.Data[0:3])
	if hdr != "---" {
		return nil
	}

	headerParts := strings.Split(string(file.Data), "---")
	var metadata HenryFileMetadata
	if _, err := toml.Decode(headerParts[1], &metadata); err != nil {
		return errors.New(fmt.Sprintf("error parsing metadata in '%s': %s", file.Name, err))
	}

	file.HasMetadata = true
	file.Metadata = &metadata

	return nil
}
