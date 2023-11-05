package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"slices"
)

// here we are specifying the maximum number of go routines we want to have running
// and keep track of how many are currently running
var max_routines = 15
var current_running = 0
var root = ""

type Excude struct {
	Exclude []string `json:"exclude"`
}

// search_mode can bee one of 3 values
//
//	1: the program matches to the name of a file only
//	2: the program matches to the extention of a file only
//	3: the program matches to the name and extention of a file
var search_mode = 1
var search_term = ""
var term_ext = ""
var outputs []string
var exclusion Excude

func compare_to_search(path string, file fs.FileInfo, outputs *[]string) {
	switch search_mode {
	case 1:
		file_name := strings.Split(file.Name(), ".")[0]
		if strings.ToLower(file_name) == strings.ToLower(search_term) {
			full_path := path + file.Name()
			if file.IsDir() {
				full_path += "/"
			}
			*outputs = append(*outputs, full_path)
		}
	case 2:
		if strings.ToLower(filepath.Ext(file.Name())) == strings.ToLower(term_ext) {
			full_path := path + file.Name() + filepath.Ext(file.Name())
			*outputs = append(*outputs, full_path)
		}
	case 3:
		file_name := strings.Split(file.Name(), ".")[0]
		if (strings.ToLower(filepath.Ext(file.Name())) == strings.ToLower(term_ext)) && (strings.ToLower(file_name) == strings.ToLower(search_term)) {
			full_path := path + file.Name()
			*outputs = append(*outputs, full_path)

		}
	}
}
func expand_node(cur *int, wg *sync.WaitGroup, path string, outputs *[]string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, file := range files {
			compare_to_search(path, file, outputs)
			if file.IsDir() && !slices.Contains(exclusion.Exclude, file.Name()) {
				// expand and keep searching
				if *cur < max_routines {
					// run the next dir search as a goroutine
					wg.Add(1)
					*cur = *cur + 1
					go func(inst_file fs.FileInfo) {
						defer wg.Done()
						new_path := path + inst_file.Name() + "/"
						expand_node(cur, wg, new_path, outputs)
						*cur = *cur - 1
					}(file)
				} else {
					// run the next dir search recursively
					new_path := path + file.Name() + "/"
					expand_node(cur, wg, new_path, outputs)
				}
			}

		}
	}
}
func parse_input(user_input string) {
	if len(user_input) == 0 {
		fmt.Println("Search term required")
	} else {
		if user_input[0] == '.' {
			term_ext = user_input
			search_mode = 2
		} else {
			split := strings.Split(user_input, ".")
			switch len(split) {
			case 1:
				search_mode = 1
				search_term = user_input
			case 2:
				search_mode = 3
				search_term = split[0]
				term_ext = "." + split[1]
			}
		}
	}
}

func main() {
	json_file, err := os.Open("./exclusion.json")
	if err != nil {
		log.Fatal(err)
	}

	json_data, err := ioutil.ReadAll(json_file)
	json_file.Close()
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(json_data, &exclusion)
	Op_Sys := runtime.GOOS
	var wg sync.WaitGroup

	switch Op_Sys {
	case "windows":
		root = "C:\\"
	case "darwin":
		root = "/Users/"
	case "linux":
		root = "/"
	default:
		root = "/"
	}
	start := time.Now()
	parse_input(os.Args[1])
	wg.Add(1)
	current_running += 1
	go func(path string) {
		defer wg.Done()
		expand_node(&current_running, &wg, path, &outputs)
	}(root)
	wg.Wait()

	defer func() {
		for _, val := range outputs {
			fmt.Println(val)
		}
	}()

	defer fmt.Println("This search took:", time.Since(start))

}
