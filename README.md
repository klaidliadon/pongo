# PonGo

PonGo (Properties on Go) is a package that converts properties files in complex structures.

The file with the following content, for example:

```ini
map.a=11
map.b=22
name=myname
age=33
```

loaded in struct

```go
struct {
	TheName string `pongo:"name"`
	TheAge  int    `pongo:"age"`
	Map     map[string]int
}
```

becomes

```
{TheName:"myname" TheAge:33 Map:map[a:11 b:22]}
```

PonGo supports the following tags:

	timeformat: specifies a time parsing format
	inline:		the value is intended as an csv inline array 


## Special Types

### Time

A time.Time field is parsed with the following timeformat `2006-01-02 15:04:05`, unless the `timeformat` tag is specified.

### Map

A map includes every property that starts with its prefix. The struct

```go
struct {
	Map map[string]string
}
```

with the file

```ini
map.a = asd
map.b = lol
```
will become

	{["a":"asd" "b":"lol"]}

### Slices

A slice element without `inline` tag finds the properties with numbers after the slice prefix. The struct

```go
struct {
	Array []string
}
```

with the file

```ini
array.1 = asd
array.2 = lol
```

will become

```
{["asd" "lol"]}
```

## Environments

PonGo offers an environment feature: it works by simply adding **@environment** at the end of the property name. 
Pongo will search for the property with the environment appendix first, if nothing is found it will search the value
in the simple property.

Take the following file:

```
sampleprop@env1 = sampleValue
sampleprop = anotherValue
anotherprop@env2 = a property
anotherprop = the value
```

Loading the properties with the environment **env1** will set the properties to:

```ini
sampleprop = sampleValue
anotherprop = the value
```

Using **env2**:

```ini
sampleprop = anotherValue
anotherprop = a property
```

And without any environment

```ini
sampleprop = anotherValue
anotherprop = the value
```

See [documentation](https://godoc.org/github.com/klaidliadon/pongo) for help.

