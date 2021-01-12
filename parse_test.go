package fasthl7

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func loadFixture() ([]byte, error) {
	b, err := ioutil.ReadFile("./oru.hl7")
	if err != nil {
		return nil, err
	}

	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\r")), nil
}

func TestParser(t *testing.T) {
	b, err := loadFixture()
	if err != nil {
		t.Fatal(err)
	}

	_, out := parse(b)

	c := 0
	for tok := range out {
		if tok.typ == tokSegEnd {
			c++
		}

		if tok.typ == tokEOF {
			msg, ok := tok.val.(Message)
			if !ok {
				t.Log("invalid message")
				t.Fail()
			}

			spew.Dump(msg)
		}
	}

	if c != 16 {
		t.Logf("expected 16 segments, got %d", c)
		t.Fail()
	}
}
