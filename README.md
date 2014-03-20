#PonGo

PonGo (Properties on Go) is a package that converts properties files in complex structures.

The file with the following content, for example

	map.a=11
	map.b=22
	name=myname
	age=33

loaded in struct

	struct {
		TheName string `pongo:"name"`
		TheAge  int    `pongo:"age"`
		Map     map[string]int
	}

becomes

	{TheName:"myname" TheAge:33 Map:map[a:11 b:22]}

##Environments

PonGo offers an environment feature: it works by simply adding **@environment** at the end of the property name. 
Pongo will search for the property with the environment appendix first, if nothing is found it will search the value
in the simple property.

Take the following file

	sampleprop@env1 = sampleValue
	sampleprop = anotherValue
	anotherprop@env2 = a property
	anotherprop = the value

Loading the properties with the environment **env1** will set the properties to

	sampleprop = sampleValue
	anotherprop = the value

Using **env2**

	sampleprop = anotherValue
	anotherprop = a property

And without any environment

	sampleprop = anotherValue
	anotherprop = the value

See [![GoDoc](https://godoc.org/github.com/klaidliadon/pongo?status.png)](https://godoc.org/github.com/klaidliadon/pongo) for help

