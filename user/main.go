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
	api := rest.New().WithItem(userStore)
	http.ListenAndServe("localhost:8000", api)
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
