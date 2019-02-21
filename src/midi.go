package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"encoding/hex"
	"strings"
	"strconv"
)

// List of valuable constants
// Fixed Header Byte size
const GoHeader = 4
// Fixed Header Length Byte Size
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

// Extracts all the information from the "Header Chunk"
// Returns valuable information as a k-tuple
func extractHeader(encodedArray [][]string) ([][] string, string){
	// Throw away MThd
	resultArr := encodedArray[GoHeader:]

	// Grab the "length" of "Data" usually 6 as a String
	lengthHexString := sliceToHexString(combineSlice(resultArr[: GoLength])) //Can parse this to be an int
	// With this ----> strconv.ParseInt(lengthHexString, 0, 32)
	// That way we can then slice out the next bytes ---> encodedArray[:length]

	return resultArr, lengthHexString
}

// Joins together a list of byte string [34 , FC, 49 ....] into on Hex String
// appends it to "0x" which says this is a hex number to parsers
func sliceToHexString(slice []string)(string){
	return "0x" + strings.Join(slice, "")
}

// Take a list of lists or slice of slices and puts them into one list
// [[FD] , [34], [A0], ... ] -> [FD, 34, A0, ...]
func combineSlice(slice [][] string) ([]string){
	resultSlice := make([]string,0)

	for _, element := range slice {
		resultSlice = append(resultSlice, element[0])
	}

	return resultSlice
}