package main

import (
//	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"encoding/hex"
	"strings"
	"strconv"
)

// List of valuable constants
const GoHeader = 4
const GoLength  = 4


func main() {
	f, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("ioutil.ReadFile failed:", err)
	}


	// Turn file in to hex string and split by every byte [[FF], ...]
	encodedString := hex.EncodeToString(f)
	encodedArray := regexp.MustCompile("..?").FindAllStringSubmatch(encodedString, -1)

	// Extract information from header of MIDI file
	encodedArray, length := extractHeader(encodedArray)

	fmt.Println(encodedArray)
	d, _ := strconv.ParseInt("0x" + length, 0, 32)
	fmt.Println(d)
	fmt.Println(length)
}

func extractHeader(encodedArray [][]string) ([][] string, string){
	// Throw away MThd
	resultArr := encodedArray[GoHeader:]

	// Grab the "length" of "Data" usually 6
	length := strings.Join(combineSliceToHex(resultArr[: GoLength]), "")

	return resultArr, length
}

func combineSliceToHex(slice [][] string) ([]string){
	resultSlice := make([]string,0)

	for _, element := range slice {
		resultSlice = append(resultSlice, element[0])
	}

	return resultSlice
}