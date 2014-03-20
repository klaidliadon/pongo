/*
PonGo (Properties on Go) is a package that converts properties files in complex structures.

The file with the following content, for example
	map.a=11
	map.b=22
	name=myname
	age=33
	arr=1,2,3,4
	t=2014-03-18
loaded in struct
	struct {
		TheName string     `pongo:"name"`
		TheAge  int        `pongo:"age"`
		Map     map[string]int
		Array   []int      `pongo:"arr,inline"`
		T       *time.Time `pongo:",timeformat=2006-01-02`
	}
becomes
	{TheName:"myname" TheAge:33 Map:map[a:11 b:22] Array:[1 2 3 4] T:2014-03-18 00:00:00 +0000 UTC}

PonGo supports the following tags:
	timeformat: specifies a time parsing format
	inline: the string value in intended as an inline array splitted with the decoder separator

A prefix is a the used in the decoding phase to complete the properties names.
*/
package pongo

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Some data in the reader has not been assigned.
type ErrDataLeft struct {
	d []string
}

func (err *ErrDataLeft) Error() string {
	return fmt.Sprintf("keys unread : %s", err.d)
}

// Checks if err is a ErrDataLeft and returns the unread keys.
func IsDataLeft(err error) ([]string, bool) {
	dl, ok := err.(*ErrDataLeft)
	if !ok {
		return nil, false
	}
	return dl.d, true
}

// Unmarshal parses the properties-encoded data and stores the result in the value pointed to by v.
// It uses the default decoder.
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

// NewDecoder returns a new decoder that reads from r, env is the preferred environment, and sep is the regex used to split inline array.
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

// Decode populate v with the values extraced from decode's reader.
func (d *Decoder) Decode(v interface{}, prefix string) (err error) {
	s := decodStatus{}
	err = s.readMap(d.r)
	if err != nil {
		return err
	}
	t := reflect.TypeOf(v)
	r := reflect.ValueOf(v)
	if t.Kind() != reflect.Ptr || r.IsNil() {
		return errors.New("pointer needed")
	}
	if prefix == "" {
		prefix = "."
	}
	err = d.decodeElement(&s, r.Elem(), t.Elem(), tag{Name: prefix})
	if err != nil {
		return err
	}
	u := s.getUnread()
	if len(u) != 0 {
		return &ErrDataLeft{u}
	}
	return nil
}

func (d *Decoder) decodeElement(s *decodStatus, v reflect.Value, t reflect.Type, tg tag) (err error) {
	if tg.Name == "-" {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		return d.decodePtr(s, v, t, tg)
	}
	if tg.Name != "." {
		s.Push(tg.Name)
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("panic in %v: %s", s.pref, v)
			} else {
				s.Pop()
			}
		}()
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

func (d *Decoder) decodeSlice(s *decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
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

func (d *Decoder) decodeStruct(s *decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
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
			fv := v.Field(i)
			if !fv.CanSet() {
				continue
			}
			err := d.decodeElement(s, fv, f.Type, newTag(f))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Decoder) decodeMap(s *decodStatus, v reflect.Value, t reflect.Type) error {
	if t.Key().Kind() != reflect.String {
		return errors.New("bad map key")
	}
	v.Set(reflect.MakeMap(t))
	p := strings.Join(s.pref, ".") + "."
	for key := range s.data {
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

func (d *Decoder) decodePtr(s *decodStatus, v reflect.Value, t reflect.Type, tg tag) error {
	nv := reflect.New(v.Type().Elem())
	err := d.decodeElement(s, nv.Elem(), nv.Elem().Type(), tg)
	if err != nil {
		return err
	}
	v.Set(nv)
	return nil
}
