package mr

import (
	"crypto/rand"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
)

func pseudo_uuid() (uuid string) {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

func RemoveFiles(filePattern string) {
	directory := "." // current directory
	matchingFiles, err := filepath.Glob(filepath.Join(directory, filePattern))

	if err != nil {
		// Handle the error if there is an issue with the pattern or directory
		fmt.Println("RemoveFiles Error:", err)
		return
	}

	for _, filePath := range matchingFiles {
		err := os.Remove(filePath)
		if err != nil {
			fmt.Println("Error deleting file:", err)
		} else {
			fmt.Println("RemoveFiles deleted:", filePath)
		}
	}
}

func CreateTempFile() (f *os.File, err error) {
	tempFile, err := ioutil.TempFile(".", "mr-tmp-*")

	if err != nil {
		return nil, err
	}

	return tempFile, nil
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// Debug tracer
var isDebug bool = false

func Tracef(format string, a ...interface{}) (n int, err error) {
	if isDebug {
		return fmt.Printf(format, a...)
	}
	return 0, nil
}
