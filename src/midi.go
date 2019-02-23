package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

var trackEnd = "ff2f00"

// FILO, uses a simple array to simulate a stack
// Mutable and weak implementation, ONLY USE FUNCTIONS TO MANIPULATE
type NoteStack struct {
	s []PartialNote
}

// How big is the stack
func (ns *NoteStack) length() int {
	return len(ns.s)
}

// adds element to the tail of slice || top of stack
// [a0, a1, a2, ...., an-1], an -> [a0, a1, a2, ...., an-1, an]
func (ns *NoteStack) push(pnote PartialNote) {
	ns.s = append(ns.s, pnote)
}

// returns a partial note of the top of the stack
func (ns *NoteStack) pop() PartialNote {
	var item PartialNote

	if ns.length() == 0 {
		fmt.Println("Nothing to pop")
	} else {
		item = ns.s[len(ns.s)-1]
		ns.s = ns.s[:len(ns.s)-1]
	}

	return item
}

// End of Stack

// A Note that does not have its ending
type PartialNote struct {
	start          int // only the start of a note
	midiNumber     int // (A|B|C ..)(octave) needed for searching
	track, channel int
}

// Compare Interface of PartialNotes
func (pn *PartialNote) equals(other PartialNote) bool {
	return other.midiNumber == pn.midiNumber
}

// A full note that has a beginning tick and ending tick
type Note struct {
	start, end     int // The start and ending ticks to determine duration and order
	midiNumber     int // (A|B|C ..)(octave) needed for searching
	track, channel int //octave seem synonymous with the pitch
	// Comparing it to B3 and C4, which are simultaneous events, B3 is always before C4
}

// Uses the Ranking system on the project sight
func (n *Note) compare(other Note) int {
	comp := 1
	if n.start == other.start {
		if n.midiNumber == other.midiNumber {
			if n.track == other.track {
				if n.channel == other.channel {
					comp = 0
				} else if n.channel < other.channel {
					comp = -1
				}
			} else if n.track < other.track {
				comp = -1
			}
		} else if n.midiNumber < other.midiNumber {
			comp = -1
		}
	} else if n.start > other.start {
		comp = -1
	}

	return comp
}

// Goes into midiNote map and takes the note = index 0
func (n *Note) getNote() string {
	// map -> [Note, Octave]
	return midiNote[n.midiNumber][0]
}

// Goes into midiNote map and takes the octave = index 1
func (n *Note) getOctave() int {
	// map -> [Note, Octave]
	octave, _ := strconv.Atoi(midiNote[n.midiNumber][1])
	return octave
}

// Gives the difference between the start and end times
func (n *Note) duration() int {
	return n.end - n.start
}

// Using the template format on project site constructs a string with the following format
//ticks     note octave duration track channel
func (n *Note) String() string {
	return fmt.Sprintf("%07d:  %-4s %-5d     %05d    %02d      %02d",
		n.start, n.getNote(), n.getOctave(), n.duration(), n.track, n.channel)
}

var notesOnStack NoteStack     //Holds Partial Notes
var notesOnContainer NoteStack //Holds Partial Notes while search stack for note to turn off

// ByAge implements sort.Interface for []Person based on
// the Age field.
type ByRank []Note

func (a ByRank) Len() int           { return len(a) }
func (a ByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRank) Less(i, j int) bool { return a[i].compare(a[j]) >= 0 }

func main() {
	notesOnStack = NoteStack{make([]PartialNote, 0)}
	notesOnContainer = NoteStack{make([]PartialNote, 0)}

	// A list of notes'
	var notes = make([]Note, 0)

	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	//Get hex string comprised of file contents
	hexMIDI := hex.EncodeToString(file)
	tracks, format := getTracks(hexMIDI)

	fmt.Println("ticks     note octave duration track channel")
	for i := 0; i < len(tracks); i++ {
		trackNumber := i + 1
		if format == 1 {
			trackNumber++
		}
		notes = append(notes, getEvents(tracks[i], trackNumber)...)
	}

	sort.Sort(ByRank(notes))

	for _, note := range notes {
		// index is the index where we are
		// element is the element from someSlice for where we are
		fmt.Println(note.String())
	}

}

func getEvents(track string, trackNumber int) []Note {
	// A list of notes'
	var notes = make([]Note, 0)

	track = track[8*2:] //the first four bytes are the track header and don't contain events
	//We only care about note on and off
	currentTime := 0
	deltaTime, length := getVariableLengthNumber(track)
	currentTime += deltaTime
	track = track[length*2:]
	eventData, eventLength := getEvent(track)
	for eventData != trackEnd {
		track = track[eventLength*2:]
		if eventLength == 3 {
			firstChar := eventData[0:1]
			if firstChar == "9" {
				noteOn(currentTime, eventData, trackNumber)
			} else if firstChar == "8" {
				n := noteOff(currentTime, eventData, trackNumber)
				notes = append(notes, n)
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

	return notes
}

func stripNoteMeta(data string) (int, int) {
	// Take bit 1 and + 1 to get channel number
	x, _ := strconv.ParseInt(data[1:2], 10, 32)
	channel := x + 1

	// Get MIDI Note number
	midiNumber, _ := strconv.ParseUint(data[2:4], 16, 32)

	return int(channel), int(midiNumber)
}

func noteOff(currentTime int, data string, trackNumber int) Note {
	// Create a partial note for the end
	channel, midiNumber := stripNoteMeta(data)
	endNote := PartialNote{currentTime, midiNumber, trackNumber, channel}

	// Go through the stack until you find a corresponding on note
	cNote := notesOnStack.pop()
	for !endNote.equals(cNote) {
		notesOnContainer.push(cNote)
		cNote = notesOnStack.pop()
	}

	note := Note{cNote.start, endNote.start, endNote.midiNumber,
		endNote.track, endNote.channel}

	// Place notes back into stack in order
	for i := 0; i < notesOnContainer.length(); i++ {
		bNote := notesOnContainer.pop()
		notesOnStack.push(bNote)
	}

	return note
}

func noteOn(currentTime int, data string, trackNumber int) {
	channel, midiNumber := stripNoteMeta(data)

	cNote := PartialNote{currentTime, midiNumber, trackNumber, channel}

	notesOnStack.push(cNote)
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

// A relation between the midi number and a given note on a keyboard (or not applicable)
var midiNote = map[int][]string{
	127: {"G", "9"},
	126: {"F#", "9"},
	125: {"F", "9"},
	124: {"E", "9"},
	123: {"D#", "9"},
	122: {"D", "9"},
	121: {"C#", "9"},
	120: {"C", "9"},
	119: {"B", "8"},
	118: {"A#", "8"},
	117: {"A", "8"},
	116: {"G#", "8"},
	115: {"G", "8"},
	114: {"F#", "8"},
	113: {"F", "8"},
	112: {"E", "8"},
	111: {"D#", "8"},
	110: {"D", "8"},
	109: {"C#", "8"},
	108: {"C", "8"},
	107: {"B", "7"},
	106: {"A#", "7"},
	105: {"A", "7"},
	104: {"G#", "7"},
	103: {"G", "7"},
	102: {"F#", "7"},
	101: {"F", "7"},
	100: {"E", "7"},
	99:  {"D#", "7"},
	98:  {"D", "7"},
	97:  {"C#", "7"},
	96:  {"C", "7"},
	95:  {"B", "6"},
	94:  {"A#", "6"},
	93:  {"A", "6"},
	92:  {"G#", "6"},
	91:  {"G", "6"},
	90:  {"F#", "6"},
	89:  {"F", "6"},
	88:  {"E", "6"},
	87:  {"D#", "6"},
	86:  {"D", "6"},
	85:  {"C#", "6"},
	84:  {"C", "6"},
	83:  {"B", "5"},
	82:  {"A#", "5"},
	81:  {"A", "5"},
	80:  {"G#", "5"},
	79:  {"G", "5"},
	78:  {"F#", "5"},
	77:  {"F", "5"},
	76:  {"E", "5"},
	75:  {"D#", "5"},
	74:  {"D", "5"},
	73:  {"C#", "5"},
	72:  {"C", "5"},
	71:  {"B", "4"},
	70:  {"A#", "4"},
	69:  {"A", "4"},
	68:  {"G#", "4"},
	67:  {"G", "4"},
	66:  {"F#", "4"},
	65:  {"F", "4"},
	64:  {"E", "4"},
	63:  {"D#", "4"},
	62:  {"D", "4"},
	61:  {"C#", "4"},
	60:  {"C", "4"},
	59:  {"B", "3"},
	58:  {"A#", "3"},
	57:  {"A", "3"},
	56:  {"G#", "3"},
	55:  {"G", "3"},
	54:  {"F#", "3"},
	53:  {"F", "3"},
	52:  {"E", "3"},
	51:  {"D#", "3"},
	50:  {"D", "3"},
	49:  {"C#", "3"},
	48:  {"C", "3"},
	47:  {"B", "2"},
	46:  {"A#", "2"},
	45:  {"A", "2"},
	44:  {"G#", "2"},
	43:  {"G", "2"},
	42:  {"F#", "2"},
	41:  {"F", "2"},
	40:  {"E", "2"},
	39:  {"D#", "2"},
	38:  {"D", "2"},
	37:  {"C#", "2"},
	36:  {"C", "2"},
	35:  {"B", "1"},
	34:  {"A#", "1"},
	33:  {"A", "1"},
	32:  {"G#", "1"},
	31:  {"G", "1"},
	30:  {"F#", "1"},
	29:  {"F", "1"},
	28:  {"E", "1"},
	27:  {"D#", "1"},
	26:  {"D", "1"},
	25:  {"C#", "1"},
	24:  {"C", "1"},
	23:  {"B", "0"},
	22:  {"A#", "0"},
	21:  {"A", "0"},
	20:  {"N/A", "99"}, // The following don't have a note associated to them yet still exist
	19:  {"N/A", "99"}, // Don't know what to do with that so I am just printing "N/A" for the note with octave 99
	18:  {"N/A", "99"}, // So that they will be sorted the lowest if played at a time x with other notes
	17:  {"N/A", "99"},
	16:  {"N/A", "99"},
	15:  {"N/A", "99"},
	14:  {"N/A", "99"},
	13:  {"N/A", "99"},
	12:  {"N/A", "99"},
	11:  {"N/A", "99"},
	10:  {"N/A", "99"},
	9:   {"N/A", "99"},
	8:   {"N/A", "99"},
	7:   {"N/A", "99"},
	6:   {"N/A", "99"},
	5:   {"N/A", "99"},
	4:   {"N/A", "99"},
	3:   {"N/A", "99"},
	2:   {"N/A", "99"},
	1:   {"N/A", "99"},
	0:   {"N/A", "99"},
}
