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

var reservedParamNames = []string{"size"}

//New creates a blank API
func New() IApi {
	return &api{
		Router:    pat.New(),
		itemStore: make(map[string]items.IStore),
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
	itemStore map[string]items.IStore
}

//Add an item to the REST-full API
func (a api) WithItem(store items.IStore) IApi {
	name := store.Name()
	if len(name) == 0 {
		panic(log.Wrapf(nil, "Add() without a name"))
	}
	if _, ok := a.itemStore[name]; ok {
		panic(log.Wrapf(nil, "Add(%s) already exists", name))
	}

	a.itemStore[name] = store

	//recreate the router to include all stores
	a.Router = pat.New()
	for name, store := range a.itemStore {
		//list with filter in GET|POST
		a.Router.Get("/"+name+"s", func(res http.ResponseWriter, req *http.Request) { a.ListHandler(res, req, name, store) })
		a.Router.Post("/"+name+"s", func(res http.ResponseWriter, req *http.Request) { a.ListHandler(res, req, name, store) })

		//individual operations:
		a.Router.Get("/"+name+"/new", func(res http.ResponseWriter, req *http.Request) { a.TmplHandler(res, req, name, store) })
		a.Router.Get("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.GetHandler(res, req, name, store) })
		a.Router.Put("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.UpdHandler(res, req, name, store) })
		a.Router.Delete("/"+name+"/{id}", func(res http.ResponseWriter, req *http.Request) { a.DelHandler(res, req, name, store) })
		a.Router.Post("/"+name, func(res http.ResponseWriter, req *http.Request) { a.AddHandler(res, req, name, store) })
	}
	a.Router.Get("/", a.UnknownHandler)
	return a
}

type itemOperResult struct {
	ID string `json:"id"`
}

func (a api) UnknownHandler(res http.ResponseWriter, req *http.Request) {
	http.Error(res, fmt.Sprintf("Unknown request %s %s", req.Method, req.URL.Path), http.StatusNotFound)
} //api.UnknownHandler()

func (a api) TmplHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodGet {
		http.Error(res, "Use HTTP method GET to retrieve the template item", http.StatusBadRequest)
		return
	}

	log.Debugf("Tmpl %s", store.Name())
	item := store.Tmpl()
	itemJSON, _ := json.Marshal(item)
	res.Write(itemJSON)
	return
} //api.TmplHandler()

func (a api) AddHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodPost {
		http.Error(res, "Use HTTP method POST to add a new item", http.StatusBadRequest)
		return
	}

	log.Debugf("Add %s", store.Name())
	newItem, err := a.BodyItem(req, store)
	if err != nil || newItem == nil {
		reason := fmt.Sprintf("Cannot process %s data from request body %s", store.Name(), err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}
	id, err := store.Add(newItem)
	if err != nil {
		reason := fmt.Sprintf("Failed to add %s %s", store.Name(), err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}

	//added - respond with new item id
	result := itemOperResult{ID: id}
	resultJSON, _ := json.Marshal(result)
	res.Write(resultJSON)
	return
} //api.AddHandler()

func (a api) GetHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodGet {
		http.Error(res, "Use HTTP method GET to retrieve an item", http.StatusBadRequest)
		return
	}

	id := req.URL.Query().Get(":id")
	log.Debugf("Get %s.id=\"%s\"", name, id)
	item, err := store.Get(id)
	if err != nil {
		log.Errorf("Cannot get id=%s %s", id, err)
		http.Error(res, "cannot get id="+id, http.StatusNotFound)
		return
	}
	itemJSON, _ := json.Marshal(item)
	res.Write(itemJSON)
	return
} //api.GetHandler()

func (a api) UpdHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodPut {
		http.Error(res, "Use HTTP method PUT to update an item", http.StatusBadRequest)
		return
	}

	id := req.URL.Query().Get(":id")
	log.Debugf("Upd %s.id=\"%s\"", name, id)

	updItem, err := a.BodyItem(req, store)
	if err != nil || updItem == nil {
		reason := fmt.Sprintf("Cannot process %s data from request body %s", store.Name(), err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}
	err = store.Upd(id, updItem)
	if err != nil {
		reason := fmt.Sprintf("Failed to update %s.id=%s %s", store.Name(), id, err)
		http.Error(res, reason, http.StatusNotAcceptable)
		return
	}
	//empty response on success
	return
} //api.UpdHandler()

func (a api) DelHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodDelete {
		http.Error(res, "Use HTTP method DELETE to delete an item", http.StatusBadRequest)
		return
	}

	id := req.URL.Query().Get(":id")
	log.Debugf("Del %s.id=\"%s\"", name, id)

	err := store.Del(id)
	if err != nil {
		reason := fmt.Sprintf("Failed to delete %s.id=%s %s", store.Name(), id, err)
		http.Error(res, reason, http.StatusNotFound)
		return
	}
	//empty response on success
	return
}

func (a api) ListHandler(res http.ResponseWriter, req *http.Request, name string, store items.IStore) {
	if req.Method != http.MethodGet && req.Method != http.MethodPost {
		http.Error(res, "Use HTTP method GET or POST to retrieve the list", http.StatusBadRequest)
		return
	}

	size, err := strconv.Atoi(req.URL.Query().Get("size"))
	if err != nil {
		size = 10
	}
	filterItem, err := a.BodyItem(req, store)
	if err != nil {
		reason := fmt.Sprintf("Cannot process %s filter from request %s", store.Name(), err)
		http.Error(res, reason, http.StatusBadRequest)
		return
	}

	itemList := store.Find(size, filterItem)
	log.Debugf("List %s.{size=%d,filter=%v} -> %d: %v", store.Name(), size, filterItem, len(itemList), itemList)

	jsonList, _ := json.Marshal(itemList)
	res.Write(jsonList)
} //api.ListHandler()

//BodyItem parses the request body as a store item
//then apply any URL parameters on top of that, overwriting the body attributes
//(it does not do validation because filter items do not have to be valid)
func (a api) BodyItem(req *http.Request, store items.IStore) (items.IItem, error) {
	//create a new item in memory
	itemDataPtrValue := reflect.New(store.Type())
	itemDataPtr := itemDataPtrValue.Interface()
	newItem, ok := itemDataPtr.(items.IItem)
	if !ok {
		return nil, log.Wrapf(nil, "failed to create new item from store.type=%v", store.Type())
	}

	//decode the (optional) request body into the item
	err := json.NewDecoder(req.Body).Decode(itemDataPtr)
	if err != nil && err != io.EOF {
		return nil, log.Wrapf(err, "request body is not valid JSON data for %s", store.Name())
	}

	// if err == nil {
	// 	log.Debugf("Item from body: %T: %+v", itemDataPtr, itemDataPtr)
	// } //if read item from request body

	//process URL parameters to overwrite item struct fields
	structType := store.Type()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	if structType.Kind() == reflect.Struct {
		//see if this param is an store item JSON attribute
		setParamCount := 0
		for index := 0; index < structType.NumField(); index++ {
			fieldType := structType.Field(index)
			fieldName := fieldType.Name
			if len(fieldType.Tag.Get("json")) > 0 {
				fieldName = fieldType.Tag.Get("json")
			}
			//skip reserved names used in REST interface
			reserved := false
			for _, reservedName := range reservedParamNames {
				if fieldName == reservedName {
					log.Debugf("Not applying URL param %s with reserved name as item attr.", fieldName)
					reserved = true
					break
				}
			} //for each reserved name

			if !reserved {
				stringValue := req.URL.Query().Get(fieldName)
				if len(stringValue) > 0 {
					//field value is specified in params
					fieldValue := itemDataPtrValue.Elem().Field(index)
					if reflect.TypeOf(stringValue).AssignableTo(fieldType.Type) {
						//can assign string value
						fieldValue.Set(reflect.ValueOf(stringValue))
						setParamCount++
					} else {
						//cannot assign string value, try int
						intValue, err := strconv.Atoi(stringValue)
						if err == nil {
							//param has int value
							if reflect.TypeOf(intValue).AssignableTo(fieldType.Type) {
								//can assign int value
								fieldValue.Set(reflect.ValueOf(intValue))
								setParamCount++
							} else {
								return nil, log.Wrapf(nil, "URL param %s=%s cannot be used to filter. Only top level int and string attributes can be used", fieldName, stringValue)
							} //if cannot assign as int
						} //if got int value
					} //if cannot assign as string
				} //if value specified
			} //if not reserved name
		} //for each struct.field[]

		// if setParamCount > 0 {
		// 	log.Debugf("Item after applying attrs: %T: %+v", itemDataPtr, itemDataPtr)
		// }
	} else {
		log.Errorf("%v is not struct", structType)
	} //if has struct type for store item
	return newItem, nil
}
