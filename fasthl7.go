package fasthl7

import (
	"bytes"
	"errors"
)

const (
	SegmentSeparator = '\r'
)

type Message []Segment
type Segment []Field

func (s Segment) Name() string {
	return string(s[0][0][0][0])
}

type Field []Repetition
type Repetition []Component
type Component []Subcomponent
type Subcomponent []byte

type Delimiters struct {
	Field        byte
	Component    byte
	Repeat       byte
	Escape       byte
	Subcomponent byte
}

func GetDelimiters(msg []byte) (*Delimiters, error) {
	if len(msg) < 8 {
		return nil, errors.New("error: msg too short")
	}

	if !bytes.HasPrefix(msg, []byte("MSH")) {
		return nil, errors.New("error: missing header")
	}

	return &Delimiters{
		Field:        msg[3],
		Component:    msg[4],
		Repeat:       msg[5],
		Escape:       msg[6],
		Subcomponent: msg[7],
	}, nil
}

func ParseMessage(b []byte) (Message, error) {

	return nil, nil
}
