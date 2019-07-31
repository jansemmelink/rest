//Package main demonstrate a very simple REST server built with this package
package main

import (
	"net/http"
	"strings"

	items "github.com/jansemmelink/items2"
	"github.com/jansemmelink/items2/store/jsonfiles"
	"github.com/jansemmelink/log"
	"github.com/jansemmelink/rest"
)

func main() {
	log.DebugOn()
	userStore := jsonfiles.MustNew("./share", "user", user{})
	vehicleStore := jsonfiles.MustNew("./share", "vehicle", &vehicle{})
	api := rest.New().WithItem(userStore).WithItem(vehicleStore)
	err := http.ListenAndServe("localhost:8000", api)
	if err != nil {
		panic(err)
	}
}

type user struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (u user) Validate() error {
	if len(u.Name) == 0 {
		return log.Wrapf(nil, "user.name not specified")
	}
	if u.Age < 0 {
		return log.Wrapf(nil, "user.age is negative")
	}
	return nil
}

//Match is used to filter in a list
//it is simple, not doing range checks, just substring on name and exact value of age
//todo: add a filter struct type to support like age ranges etc...
func (u user) Match(filter items.IItem) error {
	uf := filter.(*user)
	if len(uf.Name) > 0 {
		//look for sub-match of filter anywhere in name
		if strings.Index(u.Name, uf.Name) == -1 {
			return log.Wrapf(nil, "name mismatch (%s,%s)", u.Name, uf.Name)
		}
	}
	if uf.Age != 0 {
		//look for sub-match of age
		if u.Age != uf.Age {
			return log.Wrapf(nil, "age mismatch(%d,%d)", u.Age, uf.Age)
		}
	}
	return nil
}

type vehicle struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
}

//Validate ...
func (v *vehicle) Validate() error {
	if len(v.Manufacturer) == 0 {
		return log.Wrapf(nil, "vehicle:{\"manufacturer\":\"...\"} not specified")
	}
	if len(v.Model) == 0 {
		return log.Wrapf(nil, "vehicle:{\"model\":\"...\"} not specified")
	}
	return nil
}

func (v *vehicle) ID() string {
	return v.Manufacturer + "_" + v.Model
}

func (v *vehicle) Match(filter items.IItem) error {
	if filter != nil {
		f := filter.(*vehicle)
		if len(f.Manufacturer) > 0 {
			if strings.Index(strings.ToUpper(v.Manufacturer), strings.ToUpper(f.Manufacturer)) == -1 {
				return log.Wrapf(nil, "name mismatch (%s,%s)", v.Manufacturer, f.Manufacturer)
			}
		}
		if len(f.Model) > 0 {
			if strings.Index(strings.ToUpper(v.Model), strings.ToUpper(f.Model)) == -1 {
				return log.Wrapf(nil, "name mismatch (%s,%s)", v.Model, f.Model)
			}
		}
	}
	return nil
}
