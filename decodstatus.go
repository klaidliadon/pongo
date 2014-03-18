package pongo

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type decodStatus struct {
	p []string
	d map[string]string
}

func (s *decodStatus) readMap(r io.Reader) (err error) {
	s.d = make(map[string]string)
	var l, v, lastKey = 0, "", ""
	for r := bufio.NewReader(r); err == nil; l++ {
		v, err = r.ReadString(byte('\n'))
		if v == "" || v[0] == '#' || v[0] == '\n' {
			continue
		}
		v = clean(v)
		for ; v[len(v)-1] == '\\'; l++ {
			x, err := r.ReadString(byte('\n'))
			if err != nil && err != io.EOF {
				return err
			}
			v = v[:len(v)-1] + "\n" + strings.Trim(clean(x), " \t")
		}
		if v[0] == '\t' {
			s.d[lastKey] = s.d[lastKey] + " " + v[1:]
			continue
		}
		i := strings.Index(v, "=")
		if i < 1 {
			return fmt.Errorf("bad row %v, %s", l, v)
		}
		lastKey = strings.Trim(v[:i], " ")
		s.d[lastKey] = strings.Trim(v[i+1:], " ")
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func (s *decodStatus) GetValue(env string) (string, bool) {
	p := strings.Join(s.p, ".")
	val, ok := s.d[p+"@"+env]
	if !ok {
		val, ok = s.d[p]
	}
	return val, ok
}

func (s *decodStatus) GetIndex() []int {
	indexMap := make(map[int]bool)
	p := strings.Join(s.p, ".") + "."
	for key := range s.d {
		if !strings.HasPrefix(key, p) {
			continue
		}
		keySuffix := key[len(p):]
		if point := strings.Index(keySuffix, "."); point != -1 {
			keySuffix = keySuffix[:point]
		}
		keyNum, err := strconv.Atoi(keySuffix)
		if err != nil {
			return nil
		}
		indexMap[keyNum] = true
	}
	index := make([]int, 0, len(indexMap))
	for k := range indexMap {
		index = append(index, k)
	}
	sort.Ints(index)
	return index
}

type prefix []string

func (s *decodStatus) Push(v string) {
	s.p = append(s.p, v)
}

func (s *decodStatus) Pop() string {
	v := s.p[len(s.p)-1]
	s.p = s.p[:len(s.p)-1]
	return v
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
			tg.Modifiers[v[i]] = ""
			continue
		}
		tg.Modifiers[v[i][:x]] = v[i][i+x:]
	}
	return
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
