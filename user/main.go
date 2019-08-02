//Package main demonstrate a very simple REST server built with this package
package main

import (
	"net/http"
	"strings"

	items "github.com/jansemmelink/items2"
	"github.com/jansemmelink/items2/store/jsonfile"
	"github.com/jansemmelink/items2/store/jsonfiles"
	"github.com/jansemmelink/log"
	"github.com/jansemmelink/rest"
)

func main() {
	log.DebugOn()
	api := rest.New()

	//directories with JSON files:
	userStore := jsonfiles.MustNew("./share", "user", user{})
	api = api.WithItem(userStore)

	vehicleStore := jsonfiles.MustNew("./share", "vehicle", &vehicle{})
	api = api.WithItem(vehicleStore)

	//single JSON file:
	serviceStore := jsonfile.MustNew("./conf/services.json", "service", service{})
	api = api.WithItem(serviceStore)

	routesStore := jsonfile.MustNew("./conf/routes.json", "route", &route{})
	api = api.WithItem(routesStore)

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

type service struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Oper   string `json:"oper"`
}

//Validate ...
func (v service) Validate() error {
	if len(v.Name) == 0 {
		return log.Wrapf(nil, "service:{\"name\":\"...\"} not specified")
	}
	if len(v.Domain) == 0 {
		return log.Wrapf(nil, "service:{\"domain\":\"...\"} not specified")
	}
	if len(v.Oper) == 0 {
		return log.Wrapf(nil, "service:{\"oper\":\"...\"} not specified")
	}
	return nil
}

func (v service) ID() string {
	return v.Name
}

func (v service) Match(filter items.IItem) error {
	if filter != nil {
		f := filter.(*service)
		if len(f.Name) > 0 {
			if strings.Index(strings.ToUpper(v.Name), strings.ToUpper(f.Name)) == -1 {
				return log.Wrapf(nil, "name mismatch (%s,%s)", v.Name, f.Name)
			}
		}
	}
	return nil
}

type route struct {
	Name    string `json:"name"`
	Service string `json:"service"`
}

//Validate ...
func (r *route) Validate() error {
	if len(r.Name) == 0 {
		return log.Wrapf(nil, "route:{\"name\":\"...\"} not specified")
	}
	if len(r.Service) == 0 {
		//show we can update in Validate() with pointer receiver
		r.Service = "123"
	}
	log.Debugf("Validated route:%+v", *r)
	return nil
}

func (r route) ID() string {
	return r.Name
}

func (r route) Match(filter items.IItem) error {
	if filter != nil {
		f := filter.(*route)
		if len(f.Name) > 0 {
			if strings.Index(strings.ToUpper(r.Name), strings.ToUpper(f.Name)) == -1 {
				return log.Wrapf(nil, "name mismatch (%s,%s)", r.Name, f.Name)
			}
		}
	}
	return nil
}
