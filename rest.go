package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	items "github.com/jansemmelink/items2"

	"github.com/jansemmelink/log"
)

//New creates a blank API
func New() IApi {
	return &api{
		items: make(map[string]apiItem),
	}
}

//IApi represents a REST-full API
type IApi interface {
	WithItem(store items.IStore) IApi
	//With(otherAPI IApi) IApi

	ServeHTTP(res http.ResponseWriter, req *http.Request)
}

type api struct {
	items map[string]apiItem
}

//Add an item to the REST-full API
func (a api) WithItem(store items.IStore) IApi {
	name := store.Name()
	tmpl := reflect.New(store.ItemType()).Interface()
	if len(name) < 0 {
		panic(log.Wrapf(nil, "Add(%s,%T) needs a name", name, tmpl))
	}
	if _, ok := a.items[name]; ok {
		panic(log.Wrapf(nil, "Add(%s,%T) uses duplicate name", name, tmpl))
	}
	_, ok := tmpl.(items.IItem)
	if !ok {
		panic(log.Wrapf(nil, "Add(%s,%T) does not implement IItem", name, tmpl))
	}

	a.items[name] = apiItem{name: name, tmpl: tmpl, store: store}
	return a
}

func (a api) With(otherAPI IApi) IApi {
	// if len(name) < 0 {
	// 	panic(log.Wrapf(nil, "Add(%s,%T) needs a name", name, tmpl))
	// }
	// if _, ok := a.items[name]; ok {
	// 	panic(log.Wrapf(nil, "Add(%s,%T) uses duplicate name", name, tmpl))
	// }
	// _, ok := tmpl.(IItem)
	// if !ok {
	// 	panic(log.Wrapf(nil, "Add(%s,%T) does not implement IItem", name, tmpl))
	// }
	//a.subs[name] = apiItem{name: name, tmpl: tmpl}
	return a
}

//ServeHTTP ...
func (a api) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Debugf("HTTP %s %s", req.Method, req.URL)
	if len(req.URL.Path) > 0 || req.URL.Path[0] == '/' {
		names := strings.SplitN(req.URL.Path[1:], "/", 2)
		if len(names) > 0 && len(names[0]) > 0 {
			name := names[0]
			ai, ok := a.items[name]
			if ok {
				log.Debugf("FOUND singular %s: ai=%+v", name, ai)
				a.ServeItem(res, req, req.URL.Path[len(name)+1:], ai)
				return
			} //if singular item method

			//not singular name
			//see if this is the plural name for a list
			nl := len(name)
			if nl > 1 && name[nl-1] == 's' {
				ai, ok = a.items[name[0:nl-1]]
				if ok {
					log.Debugf("FOUND plural %s: ai=%+v", name, ai)
					a.ServeItems(res, req, req.URL.Path[len(name)+1:], ai)
					return
				} //if found plural
			} //if name ends with an 's'
		} //if path has "/<name>"
	} //if path starts with '/'

	http.Error(res, "Unknown path", http.StatusNotFound)
	return
}

type itemOperResult struct {
	ID string `json:"id"`
}

func (a api) ServeItem(res http.ResponseWriter, req *http.Request, remPath string, ai apiItem) {
	//remPath is "/123" when serving "user" from URL "/user/123"
	id := ""
	if len(remPath) > 1 {
		id = remPath[1:]
	}
	log.Debugf("ServeItem(%s): id=\"%s\"", remPath, id)

	switch req.Method {
	case http.MethodGet: //GET to retrieve
		item, err := ai.store.Get(id)
		if err != nil {
			log.Errorf("Cannot get id=%s: %v", id, err)
			http.Error(res, "cannot get id="+id, http.StatusNotFound)
			return
		}
		itemJSON, _ := json.Marshal(item)
		res.Write(itemJSON)
		return

	case http.MethodPut: //PUT to create new
		if len(id) > 0 {
			http.Error(res, "id not allowed with PUT", http.StatusBadRequest)
			return
		}

		value := make(map[string]interface{})
		err := json.NewDecoder(req.Body).Decode(&value)
		if err != nil {
			reason := fmt.Sprintf("Failed to parse JSON object for %s: %v", ai.name, err)
			http.Error(res, reason, http.StatusBadRequest)
			return
		}

		newItem, err := ai.store.New(value)
		if err != nil {
			reason := fmt.Sprintf("Failed to create %s: %v", ai.name, err)
			http.Error(res, reason, http.StatusServiceUnavailable)
			return
		}

		//created:
		result := itemOperResult{ID: newItem.ID()}
		resultJSON, _ := json.Marshal(result)
		res.Write(resultJSON)
		return

	case http.MethodPost: //POST to update
		//parse new JSON value
		var value interface{}
		if err := json.NewDecoder(req.Body).Decode(&value); err != nil {
			http.Error(res, "Invalid JSON data posted", http.StatusBadRequest)
			return
		}

		existingItem, err := ai.store.Get(id)
		if err == nil {
			updatedItem, err := existingItem.Set(value)
			if err != nil {
				reason := fmt.Sprintf("Failed to set %s.id=%s: %s", ai.name, id, err)
				http.Error(res, reason, http.StatusServiceUnavailable)
				return
			}

			//updated in memory: save to file
			log.Debugf("upd: id=%s,rev=%d,v=%+v   (BUT NOT SAVED)",
				updatedItem.ID(),
				updatedItem.Rev(),
				updatedItem.Value())
			return
		} //if got existing item

		reason := fmt.Sprintf("Failed to get %s.id=%s: %v", ai.name, id, err)
		http.Error(res, reason, http.StatusServiceUnavailable)
		return

	case http.MethodDelete:
		err := ai.store.Del(id)
		if err == nil {
			log.Debugf("deleted: id=%s", id)
			return
		} //if got existing item
		reason := fmt.Sprintf("Failed to del %s.id=%s: %v", ai.name, id, err)
		http.Error(res, reason, http.StatusNotFound)
		return

	} //switch(method)
	http.Error(res, fmt.Sprintf("Expecting GET|PUT|POST method"), http.StatusMethodNotAllowed)
	return
}

func (a api) ServeItems(res http.ResponseWriter, req *http.Request, remPath string, ai apiItem) {
	if len(remPath) > 0 {
		http.Error(res, fmt.Sprintf("Unexpected path ...%s", remPath), http.StatusNotFound)
		return
	}

	//get optional size
	size, _ := strconv.Atoi(req.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	data := ai.store.Find(size)
	dataJSON, _ := json.Marshal(data)
	res.Write(dataJSON)
	return
}

type apiItem struct {
	name  string
	tmpl  interface{}
	store items.IStore
}
