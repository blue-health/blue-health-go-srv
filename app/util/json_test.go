package util

import (
	"bytes"
	"testing"
)

func TestJSONScan(t *testing.T) {
	testCases := []struct {
		in  interface{}
		out []byte
		err error
	}{
		{in: nil, out: nullJSON},
		{in: 5, out: nullJSON, err: ErrJSONInvalid},
		{in: "", out: nullJSON},
		{in: []byte{}, out: nullJSON},
		{in: `{"some":"object"}`, out: []byte(`{"some":"object"}`)},
		{in: []byte(`{"some":"object"}`), out: []byte(`{"some":"object"}`)},
	}

	for i := range testCases {
		c := testCases[i]
		j := &JSON{}
		err := j.Scan(c.in)

		if err != c.err {
			t.Fatal("Got false error scanninng", err, "with input", c.in)
		}

		jOut, err := j.MarshalJSON()
		if err != nil {
			t.Fatal("Got error marshaling", err, "with input", c.in)
		}

		if !bytes.Equal(c.out, jOut) {
			t.Fatal("Outputs don't match for input", c.in, "Expected:", c.out, "Got:", jOut)
		}
	}
}

func TestJSONValue(t *testing.T) {
	testCases := []struct {
		in        []byte
		out       []byte
		shouldErr bool
	}{
		{in: nil, out: nil},
		{in: []byte{}, out: nil},
		{in: []byte(`{obviouslyFalse:"json}`), out: []byte{}, shouldErr: true},
		{in: []byte(`{"some":"object"}`), out: []byte(`{"some":"object"}`)},
	}

	for i := range testCases {
		c := testCases[i]
		j := FromJSON(c.in)

		jOutValue, err := j.Value()
		if err != nil {
			if c.shouldErr {
				continue
			}

			t.Fatal("Got error calling Value()", err, "with input", c.out)
		}

		if jOutValue == nil {
			if c.out == nil {
				continue
			}

			t.Fatal("jOutValue is nil for input", c.in)
		}

		jOut, ok := jOutValue.([]byte)
		if !ok {
			t.Fatal("jOutValue is not []byte")
		}

		if !bytes.Equal(c.out, jOut) {
			t.Fatal("Outputs don't match for input", c.in, "Expected:", c.out, "Got:", jOut)
		}
	}
}
