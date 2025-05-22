package fileio

import (
	"fmt"
	"log"
	"os"
)

func WriteMarkdown(fullString string, filename string) {
	resultfile, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		err := resultfile.Close()
		log.Println(err)
	}()

	_, err = resultfile.Write([]byte(fullString))
	if err != nil {
		log.Println(err)
	}
}
