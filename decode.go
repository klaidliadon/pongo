// PonGo (Properties on Go) is a simple configurarion library that converts properties files in complex structures.
// For example the file with the following content:
// 	map.a=11
// 	map.b=22
// 	name=myname
// 	age=33
// loaded in struct
// 	struct {
// 		TheName string `pongo:"name"`
// 		TheAge  int    `pongo:"age"`
// 		Map     map[string]int
// 	}
// becomes
// 	{TheName:"myname" TheAge:33 Map:map[a:11 b:22]}
package pongo

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Unmarshal parses the properties-encoded data and stores the result in the value pointed to by v.
// It uses the default decoder
func Unmarshal(b []byte, v interface{}, prefix string) error {
	d := *defaultDecoder
	d.r = bytes.NewReader(b)
	return d.Decode(v, prefix)
}

// A Decoder reads and decodes properties into struct from an input stream.
type Decoder struct {
	arrSep *regexp.Regexp
	env    string
	status decodStatus
	r      io.Reader
}

//NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader, sep, env string) (d *Decoder, err error) {
	var arrSep *regexp.Regexp
	if sep == "" {
		arrSep = defaultDecoder.arrSep
	} else {
		arrSep, err = regexp.Compile(sep)
		if err != nil {
			return nil, err
		}
	}
	return &Decoder{arrSep: arrSep, env: env, r: r}, nil
}

func (d *Decoder) Decode(v interface{}, prefix string) error {
	s := decodStatus{}
	err := s.readMap(d.r)
	if err != nil {
		return err
	}
	t := reflect.TypeOf(v)
	r := reflect.ValueOf(v)
	if t.Kind() != reflect.Ptr || r.IsNil() {
		return errors.New("pointer needed")
	}
	if prefix != "" {
		s.Push(prefix)
	}
	return d.decodeStruct(s, r.Elem(), t.Elem(), tag{})
}

func (d *Decoder) decodeElement(s decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
	if tg.Name == "-" {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		return d.decodePtr(s, v, t, tg)
	}
	if tg.Name != "." {
		s.Push(tg.Name)
		defer s.Pop()
	}
	switch v.Kind() {
	case reflect.Struct:
		return d.decodeStruct(s, v, t, tg)
	case reflect.Map:
		return d.decodeMap(s, v, t)
	case reflect.Slice:
		return d.decodeSlice(s, v, t, tg)
	default:
		val, ok := s.GetValue(d.env)
		if !ok {
			return nil
		}
		return d.decodeField(v, val)
	}
}

func (d *Decoder) decodeField(v reflect.Value, val string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(val)
	case reflect.Int:
		nv, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		v.SetInt(int64(nv))
	}
	return nil
}

func (d *Decoder) decodeSlice(s decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
	if _, isInline := tg.Modifiers["inline"]; isInline {
		val, ok := s.GetValue(d.env)
		if !ok {
			return nil
		}
		elements := d.arrSep.Split(val, -1)
		slice := reflect.MakeSlice(t, 0, len(elements))
		for _, s := range elements {
			mv := reflect.New(t.Elem())
			err := d.decodeField(mv.Elem(), s)
			if err != nil {
				return err
			}
			slice = reflect.Append(slice, mv.Elem())
		}
		v.Set(slice)
		return nil
	}
	index := s.GetIndex()
	slice := reflect.MakeSlice(t, 0, len(index))
	for _, i := range index {
		mv := reflect.New(t.Elem())
		err := d.decodeElement(s, mv.Elem(), mv.Elem().Type(), tag{Name: strconv.Itoa(i), Modifiers: tg.Modifiers})
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, mv.Elem())
	}
	v.Set(slice)
	return nil
}

func (d *Decoder) decodeStruct(s decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
	switch t.String() {
	case "time.Time":
		val, ok := s.GetValue(d.env)
		if !ok {
			return nil
		}
		tf := tg.Modifiers["timeformat"]
		if tf == "" {
			tf = "2006-01-02 15:04:05"
		}
		t, err := time.Parse(tf, val)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(t))
	default:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			err := d.decodeElement(s, v.Field(i), f.Type, newTag(f))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Decoder) decodeMap(s decodStatus, v reflect.Value, t reflect.Type) error {
	if t.Key().Kind() != reflect.String {
		return errors.New("bad map key")
	}
	v.Set(reflect.MakeMap(t))
	p := strings.Join(s.p, ".") + "."
	for key := range s.d {
		if !strings.HasPrefix(key, p) {
			continue
		}
		keySuffix := key[len(p):]
		mk := reflect.ValueOf(keySuffix)
		mv := reflect.New(t.Elem())
		err := d.decodeElement(s, mv.Elem(), mv.Elem().Type(), tag{Name: keySuffix})
		if err != nil {
			return err
		}
		v.SetMapIndex(mk, mv.Elem())
	}
	return nil
}

func (d *Decoder) decodePtr(s decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
	nv := reflect.New(v.Type().Elem())
	err := d.decodeElement(s, nv.Elem(), nv.Elem().Type(), tg)
	if err != nil {
		return err
	}
	v.Set(nv)
	return nil
}
