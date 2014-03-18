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
	"bufio"
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

// Unmarshal parses the properties-encoded data and stores the result in the value pointed to by v.
// It uses the default decoder
func Unmarshal(b []byte, v interface{}, prefix string) error {
	d := *defaultDecoder
	d.r = bytes.NewReader(b)
	return d.Decode(v, prefix)
}

type decodStatus struct {
	p prefix
	d map[string]string
}

func (s decodStatus) GetValue(env string) (string, bool) {
	p := strings.Join(s.p, ".")
	val, ok := s.d[p+"@"+env]
	if !ok {
		val, ok = s.d[p]
	}
	return val, ok
}

type prefix []string

func (p *prefix) Push(s string) {
	//fmt.Println("->", p, s)
	*p = append(*p, s)
}

func (p *prefix) Pop() string {
	s := (*p)[len(*p)-1]
	//fmt.Println("<-", p, s)
	*p = (*p)[:len(*p)-1]
	return s
}

var defaultDecoder = &Decoder{arrSep: regexp.MustCompile(`\s*,\s*|\s+`)}

type tag struct {
	Name      string
	Modifiers map[string]string
}

func newTag(t reflect.StructField) (tg tag) {
	s := t.Tag.Get("pongo")
	tg.Name = strings.ToLower(t.Name)
	if s == "" {
		return
	}
	v := strings.Split(s, ",")
	if v[0] != "" {
		tg.Name = v[0]
	}
	if len(v) == 1 {
		return
	}
	tg.Modifiers = make(map[string]string)
	for i := 1; i < len(v); i++ {
		x := strings.Index(v[i], "=")
		if x < 1 {
			continue
		}
		tg.Modifiers[v[i][:x]] = v[i][i+x:]
	}
	return
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
	data, err := readMap(d.r)
	if err != nil {
		return err
	}
	t := reflect.TypeOf(v)
	r := reflect.ValueOf(v)
	if t.Kind() != reflect.Ptr || r.IsNil() {
		return errors.New("pointer needed")
	}
	s := decodStatus{d: data}
	if prefix != "" {
		s.p.Push(prefix)
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
		s.p.Push(tg.Name)
		defer s.p.Pop()
	}
	switch v.Kind() {
	case reflect.Struct:
		return d.decodeStruct(s, v, t, tg)
	case reflect.Map:
		return d.decodeMap(s, v, t)
	default:
		val, ok := s.GetValue(d.env)
		if !ok {
			return nil
		}
		if v.Kind() == reflect.Slice {
			return d.decodeSlice(v, t, val)
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

func (d *Decoder) decodeSlice(v reflect.Value, t reflect.Type, val string) error {
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

func clean(s string) string {
	if len(s) > 1 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 1 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}

func readMap(r io.Reader) (dd map[string]string, err error) {
	dd = make(map[string]string)
	var l, s, lastKey = 0, "", ""
	for r := bufio.NewReader(r); err == nil; l++ {
		s, err = r.ReadString(byte('\n'))
		if s == "" || s[0] == '#' {
			continue
		}
		s = clean(s)
		for ; s[len(s)-1] == '\\'; l++ {
			v, err := r.ReadString(byte('\n'))
			if err != nil && err != io.EOF {
				return nil, err
			}
			s = s[:len(s)-1] + "\n" + strings.Trim(clean(v), " \t")
		}
		if s[0] == '\t' {
			dd[lastKey] = dd[lastKey] + " " + s[1:]
			continue
		}
		i := strings.Index(s, "=")
		if i < 1 {
			return nil, fmt.Errorf("bad row %v, %s", l, s)
		}
		lastKey = strings.Trim(s[:i], " ")
		dd[lastKey] = strings.Trim(s[i+1:], " ")
	}
	if err != io.EOF {
		return nil, err
	}
	return dd, nil
}
