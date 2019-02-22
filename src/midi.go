package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

var trackEnd = "ff2f00"

func main() {
	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	//Get hex string comprised of file contents
	hexMIDI := hex.EncodeToString(file)
	tracks := getTracks(hexMIDI)
	for _, track := range tracks {
		getEvents(track)
	}
}

func getEvents(track string) {
	track = track[8*2:] //the first four bytes are the track header and don't contain events
	//We only care about note on and off events
	deltaTime := getDeltaTime(track)
	deltaTime++
}

/*
	Gets the variable-length delta time by implenting the algorithm described below
	http://valentin.dasdeck.com/midi/midifile.htm
	Each non-terminal byte will have a  1 as the leftmost bit.
	All bytes will ignore the left most bit
*/
func getDeltaTime(track string) int {
	index := 0
	value := getInt(track, index, index+1)
	if (value & 0x80) != 0 { //check if leftmost bit is 1
		msb := value  //if it is 1, then //Most Significant Bit
		value &= 0x7F //set it to 0
		//Check the next byte we're adding
		for msb&0x80 != 0 {
			//Add it to the current value
			index++
			msb = getInt(track, index, index+1)
			value = (value << 7) + msb&0x7F
		}
	}
	return value
}

/*
	Returns an array of hex strings containing one track per string
	If the MIDI file is in format 1, we ignore the first track
*/
func getTracks(hexMIDI string) []string {
	//Get information from the file header
	length, format, numTracks := getHeaderInfo(hexMIDI)
	fmt.Printf("Header Length: %d\nFormat: %d\nNumber of Tracks: %d\n", length, format, numTracks)
	//We'll split the rest of the hex string into the tracks
	//hexMIDI[(8+length)*2:)] gets rid of the header (length doesn't include the first 8 bytes
	//Split after N uses the track end signifier to split the string into the remaining tracks
	tracks := strings.SplitAfterN(hexMIDI[(8+length)*2:], trackEnd, numTracks)
	//Format 0 specifies that only one track should exist
	if format == 0 && len(tracks) != 1 {
		fmt.Printf("Error\n")
	} else if format == 1 {
		//Format 1 says there's one or more tracks with an intial track containing the tempo info
		//We don't need this info
		tracks = tracks[1:]
	}
	//We don't need to do anything for format 2
	return tracks
}

/* First character is index 0
 * Includes start, excludes end
 */
func substring(s string, start, end int) string {
	startIndex := start * 2
	endIndex := end * 2
	return s[startIndex:endIndex]
}

/*
	Returns the integer specified by substring of the hex string
*/
func getInt(hexString string, start, end int) int {
	num, err := strconv.ParseInt(substring(hexString, start, end), 16, 32)
	if err != nil {
		log.Fatal(err)
	}
	return int(num)
}

/*
	Returns information stored in header according to midi specifications
	https://www.csie.ntu.edu.tw/~r92092/ref/midi/
*/
func getHeaderInfo(hexString string) (int, int, int) {
	//We could use the first four bytes to check if its a valid MIDI file
	//But I'm assuming it will work
	length := getInt(hexString, 4, 8)
	format := getInt(hexString, 8, 10)
	numTracks := getInt(hexString, 10, 12)
	return length, format, numTracks
}
