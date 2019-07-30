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
	userStore := jsonfiles.MustNew("./users", "user", user{})
	api := rest.New().WithItem(userStore)
	http.ListenAndServe("localhost:8000", api)
}

type user struct {
	Name string
}

func (u user) Validate() error {
	return nil
}

func (u user) Match(filter items.IItem) error {
	uf := filter.(*user)
	if len(uf.Name) > 0 {
		//look for sub-match of filter anywhat in name
		if strings.Index(u.Name, uf.Name) == -1 {
			return log.Wrapf(nil, "name mismatch")
		}
	}
	return nil
}
