package fasthl7

import (
	"fmt"
)

type tokenType int

const (
	eof              = byte(rune(0))
	tokErr tokenType = iota
	tokSegStart
	tokSegEnd
	tokFieldStart
	tokFieldEnd
	tokRepStart
	tokRepEnd
	tokCompStart
	tokCompEnd
	tokSubStart
	tokSubEnd
	tokEscapeStart
	tokEOF
)

var tokString = [...]string{
	tokErr:         "error",
	tokSegStart:    "segment_start",
	tokSegEnd:      "segment_end",
	tokFieldStart:  "field_start",
	tokFieldEnd:    "field_end",
	tokRepStart:    "repitiion_start",
	tokRepEnd:      "repitition_end",
	tokCompStart:   "component_start",
	tokSubStart:    "subcomponent_start",
	tokEscapeStart: "escape_start",
	tokEOF:         "EOF",
}

type token struct {
	typ tokenType
	val interface{}
}

func (t token) String() string {
	switch t.typ {
	case tokEOF:
		return "EOF"
	case tokErr:
		return t.val.(error).Error()
	}

	return fmt.Sprintf("%s\n val: %v", tokString[t.typ], t.val)
}

type stateFn func(*parser) stateFn

type parser struct {
	msg []byte

	start int
	pos   int
	width int
	out   chan token

	delims *Delimiters
	ds     []byte

	message Message
	segment Segment
	field   Field
	rep     Repetition
	comp    Component
	sub     []byte
}

func parse(msg []byte) (*parser, chan token) {
	p := &parser{
		msg: msg,
		out: make(chan token),
	}

	go p.run()

	return p, p.out
}

func (p *parser) run() {
	for state := msgStart(p); state != nil; {
		state = state(p)
	}

	close(p.out)
}

func (p *parser) emit(t token) {
	p.out <- t
	p.start = p.pos
}

func (p *parser) next() byte {
	if p.pos >= len(p.msg)-1 {
		return eof
	}

	p.pos++

	return p.msg[p.pos]
}

func (p *parser) nextDelim() byte {
	for {
		b := p.next()
		for _, d := range p.ds {
			if b == d {
				return b
			}
		}
	}
}

func (p *parser) ignore() {
	p.start = p.pos
}

func (p *parser) backup() {
	p.pos--
	if p.start > p.pos {
		p.start = p.pos
	}
}

func (p *parser) current() byte {
	return p.msg[p.pos]
}

func (p *parser) peek() byte {
	n := p.next()
	p.backup()

	return n
}

func (p *parser) accept(valid byte) bool {
	if valid == p.next() {
		return true
	}

	p.backup()
	return false
}

func (p *parser) acceptRun(valid byte) {
	for valid == p.next() {
	}

	p.backup()
}

func (p *parser) errorf(f string, args ...interface{}) stateFn {
	p.out <- token{
		tokErr,
		[]byte(fmt.Sprintf(f, args...)),
	}

	return nil
}

func (p *parser) commitBuffer(force bool) {
	if p.sub != nil || force {
		p.comp = append(p.comp, Subcomponent(p.sub))
		p.sub = nil
	}
}

func (p *parser) commitComp(force bool) {
	p.commitBuffer(false)

	if p.comp != nil || force {
		p.rep = append(p.rep, p.comp)
		p.comp = nil
	}
}

func (p *parser) commitRep(force bool) {
	p.commitComp(false)

	if p.rep != nil || force {
		p.field = append(p.field, p.rep)
		p.rep = nil
	}
}

func (p *parser) commitField(force bool) {
	p.commitRep(false)

	if p.field != nil || force {
		p.segment = append(p.segment, p.field)
		p.field = nil
	}
}

func (p *parser) commitSeg(force bool) {
	p.commitField(false)
	if p.segment != nil || force {
		p.message = append(p.message, p.segment)
		p.emit(token{typ: tokSegEnd, val: p.segment[0][0][0][0]})
		p.segment = nil
	}
}

func msgStart(p *parser) stateFn {
	return parseHeader
}

func parseHeader(p *parser) stateFn {
	d, err := GetDelimiters(p.msg)
	if err != nil {
		return p.errorf("error parsing header: %v", err)
	}

	p.delims = d
	p.ds = []byte{d.Escape, d.Subcomponent, d.Component, d.Repeat, d.Field, SegmentSeparator, eof}

	p.segment = Segment{
		Field{Repetition{Component{Subcomponent(p.msg[0:3])}}},
		Field{Repetition{Component{Subcomponent([]byte{p.msg[3]})}}},
		Field{Repetition{Component{Subcomponent(p.msg[4:8])}}},
	}
	p.start = 8
	p.pos = 8

	if !(p.current() == p.delims.Field) {
		return p.errorf("error: invalid header expected %q, got %q", p.delims.Field, p.peek())
	}

	return parseMsg
}

func parseMsg(p *parser) stateFn {
	nl := false
	for c := p.next(); c != eof; {
		//log.Printf("eval pos: %d val: %q", p.pos, c)
		switch c {
		case SegmentSeparator:
			if !nl {
				p.commitSeg(true)
			}
			nl = true
		case '\n':
			nl = true
		case p.delims.Field:
			nl = false
			p.commitField(true)
		case p.delims.Repeat:
			nl = false
			p.commitRep(true)
		case p.delims.Component:
			nl = false
			p.commitComp(true)
		case p.delims.Subcomponent:
			nl = false
			p.commitBuffer(true)
		default:
			nl = false
			p.sub = append(p.sub, c)
		}

		c = p.next()
	}

	if p.sub != nil {
		p.commitSeg(true)
	}

	p.emit(token{typ: tokEOF, val: p.message})
	return nil
}
