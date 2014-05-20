package pongo

import (
	"bytes"
	"testing"
	"time"
)

type Client struct {
	Id    string
	Title string
	Mode  string
	Auth  struct {
		User, Pwd string
	}
	Db struct {
		Database string
		Tables   map[string]string
		Auth     struct{ Host, Port, User, Pwd string }
	}
	Scripts []struct {
		Name    string
		Command []string `pongo:",inline"`
		Note    []string `pongo:"-"`
	}
	Job struct {
		Month   []int `pongo:",inline"`
		Day     []int `pongo:",inline"`
		Weekday []int `pongo:",inline"`
		Hour    []int `pongo:",inline"`
		Minute  []int `pongo:",inline"`
	}
}

var clientFile = `
title=Dati Bottonificio,
mode=web` + "\r" + `
auth.user=web
auth.pwd=web
db.database=creavista_bottonificio
db.tables.agenti=etl_agenti
db.tables.anacli=etl_anacli
db.tables.condpag=etl_condpag
db.tables.destdiv=etl_destdiv
db.tables.divisa=etl_divisa
db.tables.fatture=etl_fatture
db.tables.materiale=etl_materiale
db.tables.tipo=etl_tipo
db.tables.ordini=etl_ordini
db.tables.zone=etl_zone
scripts.1.name=Correzione maiuscole,
scripts.1.command=python bottonificio.py
scripts.2.name=Caricamento cubo
scripts.2.command=simple-etl bottonificio.ini
scripts.3.name=Attivazione nuovo cubo
scripts.3.command=curl -so/dev/null http://creavista.gruppo4.it/switch?key=DEMO_KEY
`

type testStruct struct {
	TheName     string `pongo:"name,tag"`
	TheAge      int    `pongo:"age"`
	Map         map[string]string
	TimeMap     map[string]time.Time
	ArrayInline []int `pongo:",inline"`
	Array       []struct {
		Name    string
		Command []string `pongo:",inline"`
	}
	Timer  *time.Time
	Timer2 time.Time `pongo:"timer2,timeformat=2006"`
	val    string
	Float  float64
	SFloat float32
	Dur    time.Duration
	Bool   bool
}

var f1 = `#comment
asd.map.a=eee=eee
asd.map.b=rrrr
asd.name=1\
	2
	3
asd.age@env=120
asd.arrayinline=1,2,3,4,5,6,7
asd.array.1.name=a name
asd.array.1.command=i am,,legend
asd.array.3.name=name2
asd.array.3.command=hello,there
asd.age=33
asd.timer=2012-01-02 15:04:05
asd.timer2=2012
asd.timemap.a=2012-01-02 15:04:05
asd.timemap.b=2012-01-02 15:04:05
asd.timemap.c=2012-01-02 15:04:05
asd.val=aaaa
asd.val@env=aaaa
asd.float=10.5
asd.sfloat=10.5
asd.dur=010m05s
asd.bool=true
`

func TestSomeErrors(t *testing.T) {
	err := Unmarshal([]byte(f1), &struct{ Field map[int]int }{}, "")
	if err == nil {
		t.Errorf("error expected")
	}
}

func TestDecode(t *testing.T) {
	d, err := NewDecoder(nil, ",)	", "")
	if _, ok := IsDataLeft(err); ok || err == nil {
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
	err = d.Decode(&v, "asd")
	if _, ok := IsDataLeft(err); !ok {
		t.Errorf("error: %s", err)
	}
	t.Logf("Result (prefix `%s`):\n%+v", "asd", v)
}

func TestUnmarshal(t *testing.T) {
	v := testStruct{}
	err := Unmarshal([]byte(f1), &v, "asd")
	if _, ok := IsDataLeft(err); !ok {
		t.Errorf("error: %s", err)
	}
	t.Logf("Result (prefix `%s`):\n%+v\n%s", "asd", v, err)

	err = Unmarshal([]byte(f1+"\nz"), &v, "asd")
	if err == nil {
		t.Errorf("error expected")
	}
}

func TestUnmarshalConfig(t *testing.T) {
	v := Client{}
	if err := Unmarshal([]byte(clientFile), &v, ""); err != nil {
		t.Errorf("error: %s", err)
	}
	t.Logf("Result (prefix `%s`):\n%+v", "", v)
}
