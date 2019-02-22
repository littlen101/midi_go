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

// FILO, uses a simple array to simulate a stack
// Mutable and weak implementation, ONLY USE FUNCTIONS TO MANIPULATE
type NoteStack struct {
	s []PartialNote
}

// adds element to the tail of slice || top of stack
// [a0, a1, a2, ...., an-1], an -> [a0, a1, a2, ...., an-1, an]
func (ns *NoteStack) push(pnote PartialNote) {
	ns.s = append(ns.s, pnote)
}

// returns a partial note of the top of the stack
func (ns *NoteStack) pop() PartialNote {
	item := ns.s[len(ns.s)-1]
	ns.s = ns.s[:len(ns.s)-1]

	return item
}

// End of Stack

// A full note that has a beginning tick and ending tick
type Note struct {
	start, end             int    // The start and ending ticks to determine duration and order
	note                   string // A|B|C ..
	octave, track, channel int    //octave seem synonymous with the pitch
	// Comparing it to B3 and C4, which are simultaneous events, B3 is always before C4
}

type PartialNote struct {
	start                  int    // only the start of a note
	note                   string // A|B|C ..
	octave, track, channel int    // octave and note are needed for searching
}

// Gives the difference between the start and end times
func (n *Note) duration() int {
	return n.end - n.start
}

// Using the template format on project site constructs a string with the following format
//ticks     note octave duration track channel
func (n *Note) String() string {
	return fmt.Sprintf("%7d:  %-4s %5d %5d %2d %2d", n.start, n.note, n.octave, n.duration(), n.track, n.channel)
}

var notesOnStack = NoteStack{make([]PartialNote, 0)}
var notesOnContainer = NoteStack{make([]PartialNote, 0)}

func main() {
	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	//Get hex string comprised of file contents
	hexMIDI := hex.EncodeToString(file)
	tracks, format := getTracks(hexMIDI)
	for i := 0; i < len(tracks); i++ {
		trackNumber := i + 1
		if format == 1 {
			trackNumber++
		}
		getEvents(tracks[i], trackNumber)
	}
}

func getEvents(track string, trackNumber int) {
	track = track[8*2:] //the first four bytes are the track header and don't contain events
	//We only care about note on and off
	currentTime := 0
	deltaTime, length := getVariableLengthNumber(track)
	currentTime += deltaTime
	track = track[length*2:]
	eventData, eventLength := getEvent(track)
	for eventData != trackEnd {
		fmt.Println(eventData)
		track = track[eventLength*2:]
		if eventLength == 3 {
			firstChar := eventData[0:1]
			if firstChar == "8" {
				noteOn(currentTime, eventData, trackNumber)
			} else if firstChar == "9" {
				noteOff(currentTime, eventData, trackNumber)
			}
		}
		//For next event
		deltaTime, length := getVariableLengthNumber(track)
		currentTime += deltaTime
		track = track[length*2:]
		eventData, eventLength = getEvent(track)
	}
	/*
		for each event
			get delta time
			trim delta time from event index indicates the last byte within the delta time
	*/
}

func noteOff(currentTime int, data string, trackNumber int) {
	//TODO
}

func noteOn(currentTime int, data string, trackNumber int) {
	//TODO
}

func getEvent(track string) (string, int) {
	firstByte := substring(track, 0, 1)
	var eventLength int
	switch firstByte {
	case "f0":
		fallthrough
	case "f1":
		eventLength = getEventLength(track, 2)
	case "ff":
		secondByte := substring(track, 1, 2)
		switch secondByte {
		case "20":
			eventLength = 4
		case "2f":
			eventLength = 3
		case "51":
			eventLength = 6
		case "54":
			eventLength = 8
		case "58":
			eventLength = 7
		case "00":
			fallthrough
		case "59":
			eventLength = 5
		default:
			eventLength = getEventLength(track, 2)
		}
	default:
		firstChar := firstByte[0:1]
		if (firstChar == "c") || (firstChar == "d") {
			eventLength = 2
		} else {
			eventLength = 3
		}
	}
	return substring(track, 0, eventLength), eventLength
}

/*
	Returns the total number of bytes an event uses
*/
func getEventLength(event string, varOffset int) int {
	eventLength, length := getVariableLengthNumber(event[varOffset*2:])
	return eventLength + length + varOffset
}

/*
	Gets the variable-length number by implementing the algorithm described below
	http://valentin.dasdeck.com/midi/midifile.htm
	Each non-terminal byte will have a  1 as the leftmost bit.
	All bytes will ignore the left most bit

	returns the variable length number and its length in bytes
*/
func getVariableLengthNumber(track string) (int, int) {
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
	return value, index + 1
}

/*
	Returns an array of hex strings containing one track per string
	If the MIDI file is in format 1, we ignore the first track
*/
func getTracks(hexMIDI string) ([]string, int) {
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
	return tracks, format
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
