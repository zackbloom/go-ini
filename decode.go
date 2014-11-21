// Decode INI files with a syntax similar to JSON decoding
package ini

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal parses the INI-encoded data and stores the result
// in the value pointed to by v.
func Unmarshal(data []byte, v interface{}) error {
	var d decodeState
	d.init(data)
	return d.unmarshal(v)
}

// decodeState represents the state while decoding a INI value.
type decodeState struct {
	currentPath string
	lineNum     int
	scanner     *bufio.Scanner
	savedError  error
}

type sectionTag struct {
	wildcard bool
	value    reflect.Value
	children map[string]sectionTag
}

func (d *decodeState) init(data []byte) *decodeState {

	d.lineNum = 1
	d.scanner = bufio.NewScanner(bytes.NewReader(data))
	d.savedError = nil
	return d
}

// error aborts the decoding by panicking with err.
func (d *decodeState) error(err error) {
	panic(err)
}

// saveError saves the first err it is called with,
// for reporting at the end of the unmarshal.
func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = err
	}
}

func generateMap(m map[string]sectionTag, v reflect.Value) {

	if v.Type().Kind() == reflect.Ptr {
		generateMap(m, v.Elem())
	} else if v.Kind() == reflect.Struct {
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {

			sf := typ.Field(i)
			f := v.Field(i)

			st := sectionTag{false, f, make(map[string]sectionTag)}

			m[sf.Tag.Get("ini")] = st

			if f.Type().Kind() == reflect.Struct {
				generateMap(st.children, f)
			}
		}
	} else {
		panic(fmt.Sprintf("Don't handle this type yet: %s", v.Kind()))
	}

}

func (d *decodeState) unmarshal(x interface{}) error {

	var parentMap map[string]sectionTag = make(map[string]sectionTag)

	generateMap(parentMap, reflect.ValueOf(x))

	var parentSection sectionTag
	var hasParent bool = false

	for d.scanner.Scan() {
		line := strings.TrimSpace(d.scanner.Text())
		log.Printf("Scanned (%d): %s\n", d.lineNum, line)
		d.lineNum = d.lineNum + 1

		if len(line) < 1 || line[0] == ';' || line[0] == '#' {
			continue // skip comments
		}

		if line[0] == '[' && line[len(line)-1] == ']' {
			parentSection, hasParent = parentMap[line]
			continue
		}

		if hasParent {
			matches := strings.SplitN(line, "=", 2)

			if len(matches) == 2 {
				prop := strings.TrimSpace(matches[0])
				data := strings.TrimSpace(matches[1])

				childSection, hasChild := parentSection.children[prop]
				if hasChild {
					// set value
					//log.Println("**** Matches", matches[0], " ::: ", childSection)
					setValue(childSection.value, data, d.lineNum)
				} // else look for wildcard??
			}
		} else {
			log.Println("Look for top level Property")
		}
	}

	return nil
}

func setValue(v reflect.Value, s string, lineNum int) {
	log.Printf("SET(%s, %s)", v.Kind(), s)
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)

	case reflect.Bool:
		v.SetBool(getBoolValue(s))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v.OverflowInt(n) {
			panic(fmt.Sprintf("Invalid number '%s' specified on line %d", s, lineNum))
		}
		v.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil || v.OverflowUint(n) {
			panic(fmt.Sprintf("Invalid number '%s' specified on line %d", s, lineNum))
		}
		v.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil || v.OverflowFloat(n) {
			panic(fmt.Sprintf("Invalid number '%s' specified on line %d", s, lineNum))
		}
		v.SetFloat(n)

	default:
		log.Println("Can't set that kind yet!")
	}

}

func getBoolValue(s string) bool {
	v := false
	switch strings.ToLower(s) {
	case "t", "true", "y", "yes", "1":
		v = true
	}

	return v
}

/*
// A Decoder reads and decodes JSON objects from an input stream.
type Decoder struct {
	d    decodeState
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of JSON into a Go value.
func (dec *Decoder) Decode(v interface{}) error {
	if dec.err != nil {
		return dec.err
	}

	n, err := dec.readValue()
	if err != nil {
		return err
	}

	// Don't save err from unmarshal into dec.err:
	// the connection is still usable since we read a complete JSON
	// object from it before the error happened.
	dec.d.init(dec.buf[0:n])
	err = dec.d.unmarshal(v)

	// Slide rest of data down.
	rest := copy(dec.buf, dec.buf[n:])
	dec.buf = dec.buf[0:rest]

	return err
}
*/
