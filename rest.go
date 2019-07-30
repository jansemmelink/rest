package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	items "github.com/jansemmelink/items2"

	"github.com/gorilla/pat"
	"github.com/jansemmelink/log"
)

//New creates a blank API
func New() IApi {
	return &api{
		Router: pat.New(),
		items:  make(map[string]apiItem),
	}
}

//IApi represents a REST-full API
type IApi interface {
	WithItem(store items.IStore) IApi
	//With(otherAPI IApi) IApi

	ServeHTTP(res http.ResponseWriter, req *http.Request)
}

type api struct {
	*pat.Router
	items map[string]apiItem
}

//Add an item to the REST-full API
func (a api) WithItem(store items.IStore) IApi {
	name := store.Name()
	if len(name) == 0 {
		panic(log.Wrapf(nil, "Add() without a name"))
	}
	if _, ok := a.items[name]; ok {
		panic(log.Wrapf(nil, "Add(%s) already exists", name))
	}
	//list with filter in GET|POST
	a.Router.Get("/"+name+"s", func(res http.ResponseWriter, req *http.Request) { a.ListHandler(res, req, name, store) })
	a.Router.Post("/"+name+"s", func(res http.ResponseWriter, req *http.Request) { a.ListHandler(res, req, name, store) })

	//individual operations:
	a.Router.Get("/"+name+"/new", func(res http.ResponseWriter, req *http.Request) { a.TmplHandler(res, req, name, store) })
	a.Router.Get("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.GetHandler(res, req, name, store) })
	a.Router.Put("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.UpdHandler(res, req, name, store) })
	a.Router.Delete("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.DelHandler(res, req, name, store) })
	a.Router.Post("/"+name, func(res http.ResponseWriter, req *http.Request) { a.AddHandler(res, req, name, store) })

	a.Router.Get("/", a.UnknownHandler)
	return a
}

type itemOperResult struct {
	ID string `json:"id"`
}

func (a api) UnknownHandler(res http.ResponseWriter, req *http.Request) {
	log.Debugf("?...")
	http.Error(res, "Unknown path", http.StatusNotFound)
}

func (a api) TmplHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	log.Debugf("Tmpl...")
	item := store.Tmpl()
	itemJSON, _ := json.Marshal(item)
	res.Write(itemJSON)
	return
}

func (a api) AddHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	newItem, err := a.BodyItem(req, store)
	if err != nil || newItem == nil {
		reason := fmt.Sprintf("Cannot process %s data from request body: %v", store.Name(), err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}
	id, err := store.Add(newItem)
	if err != nil {
		reason := fmt.Sprintf("Failed to add %s: %v", store.Name(), err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}

	//added - respond with new item id
	result := itemOperResult{ID: id}
	resultJSON, _ := json.Marshal(result)
	res.Write(resultJSON)
	return
} //NewHandler()

func (a api) GetHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	id := req.URL.Query().Get(":id")
	log.Debugf("Get(%s): id=\"%s\"", name, id)
	item, err := store.Get(id)
	if err != nil {
		log.Errorf("Cannot get id=%s: %v", id, err)
		http.Error(res, "cannot get id="+id, http.StatusNotFound)
		return
	}
	itemJSON, _ := json.Marshal(item)
	res.Write(itemJSON)
	return
}

func (a api) UpdHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	// id := req.URL.Query().Get(":id")
	// log.Debugf("Upd(%s): id=\"%s\"", name, id)
	// 	//parse new JSON value
	// 	var value interface{}
	// 	if err := json.NewDecoder(req.Body).Decode(&value); err != nil {
	// 		http.Error(res, "Invalid JSON data posted", http.StatusBadRequest)
	// 		return
	// 	}

	// 	existingItem, err := ai.store.Get(id)
	// 	if err == nil {
	// 		updatedItem, err := existingItem.Set(value)
	// 		if err != nil {
	// 			reason := fmt.Sprintf("Failed to set %s.id=%s: %s", ai.name, id, err)
	// 			http.Error(res, reason, http.StatusServiceUnavailable)
	// 			return
	// 		}

	// 		//updated in memory: save to file
	// 		log.Debugf("upd: id=%s,rev=%d,v=%+v   (BUT NOT SAVED)",
	// 			updatedItem.ID(),
	// 			updatedItem.Rev(),
	// 			updatedItem.Value())
	// 		return
	// 	} //if got existing item

	// 	reason := fmt.Sprintf("Failed to get %s.id=%s: %v", ai.name, id, err)
	// 	http.Error(res, reason, http.StatusServiceUnavailable)
	// 	return

}

func (a api) DelHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	// 	err := ai.store.Del(id)
	// 	if err == nil {
	// 		log.Debugf("deleted: id=%s", id)
	// 		return
	// 	} //if got existing item
	// 	reason := fmt.Sprintf("Failed to del %s.id=%s: %v", ai.name, id, err)
	// 	http.Error(res, reason, http.StatusNotFound)
	// 	return

	// } //switch(method)
	// http.Error(res, fmt.Sprintf("Expecting GET|PUT|POST method"), http.StatusMethodNotAllowed)
	// return
}

func (a api) ListHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	size, err := strconv.Atoi(req.URL.Query().Get("size"))
	if err != nil {
		size = 10
	}
	var filterItem items.IItem
	switch req.Method {
	case http.MethodPost:
		//POST parses optional body as item filter
		var err error
		filterItem, err = a.BodyItem(req, store)
		if err != nil {
			reason := fmt.Sprintf("Cannot process %s data as filter from request body: %v", store.Name(), err)
			http.Error(res, reason, http.StatusBadRequest)
			return
		}

	case http.MethodGet:
		//GET parses optional params are item fields
		//todo:...
	}

	log.Debugf("List... filter=%T=%+v", filterItem, filterItem)
	itemList := store.Find(size, filterItem)
	log.Debugf("Got %d items: %v", len(itemList), itemList)
	jsonList, _ := json.Marshal(itemList)
	res.Write(jsonList)
}

type apiItem struct {
	name string
	//tmpl  interface{}
	store items.IStore
}

func (a api) BodyItem(req *http.Request, store items.IStore) (items.IItem, error) {
	newItemDataPtrValue := reflect.New(store.Type())
	newItemDataPtr := newItemDataPtrValue.Interface()
	err := json.NewDecoder(req.Body).Decode(newItemDataPtr)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, log.Wrapf(err, "failed to parse JSON object for %s: %v", store.Name(), err)
	}
	newItem, ok := newItemDataPtr.(items.IItem)
	if !ok {
		return nil, log.Wrapf(err, "failed to convert %T to %s", newItemDataPtr, store.Name())
	}
	if err := newItem.Validate(); err != nil {
		return nil, log.Wrapf(err, "invalid %s: %v", store.Name(), err)
	}
	return newItem, nil
}
