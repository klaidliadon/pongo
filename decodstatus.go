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
	pref []string
	data map[string]string
	env  map[string]map[string]string
}

func (s *decodStatus) getUnread() (u []string) {
	if len(s.data) != 0 {
		for k := range s.data {
			u = append(u, k)
		}
	}
	for i := range s.env {
		if len(s.env[i]) != 0 {
			for k := range s.env[i] {
				u = append(u, k+"@"+i)
			}
		}
	}
	return
}

func (s *decodStatus) readMap(r io.Reader) (err error) {
	s.data = make(map[string]string)
	s.env = make(map[string]map[string]string)
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
			s.data[lastKey] = s.data[lastKey] + " " + v[1:]
			continue
		}
		i := strings.Index(v, "=")
		if i < 1 {
			i = strings.Index(v, ":")
			if i < 1 {
				return fmt.Errorf("bad row %v, %s", l, v)
			}
		}
		lastKey = strings.Trim(strings.Replace(v[:i], "-", "", -1), " ")
		var env = ""
		if x := strings.Split(lastKey, "@"); len(x) > 1 {
			lastKey = x[0]
			env = x[1]
		}
		if env == "" {
			s.data[lastKey] = strings.Trim(v[i+1:], " ")
		} else {
			if s.env[env] == nil {
				s.env[env] = make(map[string]string)
			}
			s.env[env][lastKey] = strings.Trim(v[i+1:], " ")
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func (s *decodStatus) GetValue(env string) (string, bool) {
	key := strings.Join(s.pref, ".")
	value, exist := s.data[key]
	if e, ok := s.env[env]; ok {
		if v, ok := e[key]; ok {
			value, exist = v, ok
		}
	}
	delete(s.data, key)
	for i := range s.env {
		delete(s.env[i], key)
	}
	return value, exist
}

func (s *decodStatus) GetIndex() []int {
	indexMap := make(map[int]bool)
	p := strings.Join(s.pref, ".") + "."
	for key := range s.data {
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
	s.pref = append(s.pref, v)
}

func (s *decodStatus) Pop() string {
	v := s.pref[len(s.pref)-1]
	s.pref = s.pref[:len(s.pref)-1]
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
