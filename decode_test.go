package pongo

import (
	"bytes"
	"testing"
	"time"
)

type testStruct struct {
	TheName string `pongo:"name,tag"`
	TheAge  int    `pongo:"age"`
	Map     map[string]string
	TimeMap map[string]time.Time
	Array   []int
	Timer   *time.Time
	Timer2  time.Time `pongo:"timer2,timeformat=2006"`
}

var f1 = `#comment
asd.map.a=eee=eee
asd.map.b=rrrr
asd.name=1\
	2
	3
asd.age@env=120
asd.array=1,2,3,4,5,6,7
asd.age=33
asd.timer=2012-01-02 15:04:05
asd.timer2=2012
asd.timemap.a=2012-01-02 15:04:05
asd.timemap.b=2012-01-02 15:04:05
asd.timemap.c=2012-01-02 15:04:05
`

func TestDecode(t *testing.T) {
	d, err := NewDecoder(nil, ",)	", "")
	if err == nil {
		t.Errorf("error expected")
	}

	d, err = NewDecoder(bytes.NewReader([]byte(f1)), "", "env")
	if err != nil {
		t.Errorf("error: %s", err)
	}

	v := testStruct{}
	if d.Decode(v, "asd") == nil {
		t.Errorf("error expected")
	}

	d, err = NewDecoder(bytes.NewReader([]byte(f1)), "", "env")
	v = testStruct{}
	if err := d.Decode(&v, "asd"); err != nil {
		t.Errorf("error: %s", err)
	}
	t.Logf("Result (prefix `%s`):\n%+v", "asd", v)
}

func TestUnmarshal(t *testing.T) {
	v := testStruct{}
	if err := Unmarshal([]byte(f1), &v, "asd"); err != nil {
		t.Errorf("error: %s", err)
	}
	t.Logf("Result (prefix `%s`):\n%+v", "asd", v)

	err := Unmarshal([]byte(f1+"\nz"), &v, "asd")
	if err == nil {
		t.Errorf("error expected")
	}
}
