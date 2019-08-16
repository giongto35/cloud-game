package sdp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

//ExtMap represents the activation of a single RTP header extension
type ExtMap struct {
	Value     int
	Direction Direction
	URI       *url.URL
	ExtAttr   *string
}

//Clone converts this object to an Attribute
func (e *ExtMap) Clone() Attribute {
	return Attribute{Key: "extmap", Value: e.string()}
}

//Unmarshal creates an Extmap from a string
func (e *ExtMap) Unmarshal(raw string) error {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("SyntaxError: %v", raw)
	}

	fields := strings.Fields(parts[1])
	if len(fields) < 2 {
		return fmt.Errorf("SyntaxError: %v", raw)
	}

	valdir := strings.Split(fields[0], "/")
	value, err := strconv.ParseInt(valdir[0], 10, 64)
	if (value < 1) || (value > 246) {
		return fmt.Errorf("SyntaxError: %v -- extmap key must be in the range 1-256", valdir[0])
	}
	if err != nil {
		return fmt.Errorf("SyntaxError: %v", valdir[0])
	}

	var direction Direction
	if len(valdir) == 2 {
		direction, err = NewDirection(valdir[1])
		if err != nil {
			return err
		}
	}

	uri, err := url.Parse(fields[1])
	if err != nil {
		return err
	}

	if len(fields) == 3 {
		tmp := fields[2]
		e.ExtAttr = &tmp
	}

	e.Value = int(value)
	e.Direction = direction
	e.URI = uri
	return nil
}

//Marshal creates a string from an ExtMap
func (e *ExtMap) Marshal() string {
	return attributeKey + e.Name() + ":" + e.string() + endline
}

func (e *ExtMap) string() string {
	output := fmt.Sprintf("%d", e.Value)
	dirstring := e.Direction.String()
	if dirstring != directionUnknownStr {
		output += "/" + dirstring
	}

	if e.URI != nil {
		output += " " + e.URI.String()
	}

	if e.ExtAttr != nil {
		output += " " + *e.ExtAttr
	}

	return output
}

//Name returns the constant name of this object
func (e *ExtMap) Name() string {
	return "extmap"
}
