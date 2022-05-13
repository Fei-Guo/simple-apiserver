package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"

	"github.com/gorilla/mux"
	"github.com/simple-apiserver/api"
)

// The Store maintains a simple in-memory db. The key is the application title,
// and all revision histories will be recorded.
// The db is protected by a single mutex and will be compacted periodically.

type Store struct {
	db map[string][]*api.Application
	sync.Mutex
}

var appDB *Store

func NewStore() *Store {
	return &Store{
		db: make(map[string][]*api.Application),
	}
}

var router = mux.NewRouter()

// To be extended
var supportedQueries = map[string]struct{}{
	"company": struct{}{},
	"source":  struct{}{},
}

func InitInMemDB() {
	appDB = NewStore()
}

// Get returns the latest version of the key
func (s *Store) Get(key string) (*api.Application, error) {
	s.Lock()
	defer s.Unlock()
	if allRevs, ok := s.db[key]; ok {
		if len(allRevs) == 0 {
			panic(fmt.Sprintf("Key %s should have at least one record", key))
		}
		last := allRevs[len(allRevs)-1]
		if last.DeleteTimeStamp != nil {
			return nil, fmt.Errorf("Err: application with title %s has been deleted", key)
		}
		return api.DeepCopy(last), nil
	}
	return nil, fmt.Errorf("Err: application with title %s does not exist", key)
}

// DumpKey returns all versions of the key
func (s *Store) DumpKey(key string) ([]*api.Application, error) {
	s.Lock()
	defer s.Unlock()
	ret := make([]*api.Application, 0)

	if allRevs, ok := s.db[key]; ok {
		if len(allRevs) == 0 {
			panic(fmt.Sprintf("Key %s should have at least one record", key))
		}
		for _, each := range allRevs {
			ret = append(ret, api.DeepCopy(each))
		}
		return ret, nil
	}
	return nil, fmt.Errorf("Err: application with title %s does not exist", key)
}

// List returns the latest versions for all matching keys
func (s *Store) List(options map[string][]string) ([]*api.Application, error) {
	s.Lock()
	defer s.Unlock()

	sources, matchSource := options["source"]
	companies, matchCompany := options["company"]
	ret := make([]*api.Application, 0)
	for k, v := range s.db {
		if len(v) == 0 {
			panic(fmt.Sprintf("Key %s should have at least one record", k))
		}
		last := v[len(v)-1]
		if last.DeleteTimeStamp != nil {
			continue
		}
		if matchSource && !inArray(last.Source, sources) {
			continue
		}
		if matchCompany && !inArray(last.Company, companies) {
			continue
		}
		// return the latest version
		ret = append(ret, api.DeepCopy(last))
	}
	return ret, nil
}

// Create adds the object to the DB if it does not exist or has been deleted before
func (s *Store) Create(app *api.Application) error {
	s.Lock()
	defer s.Unlock()

	if err := validateApplication(app); err != nil {
		return err
	}
	clone := api.DeepCopy(app)
	if allRevs, ok := s.db[app.Title]; ok {
		if len(allRevs) == 0 {
			panic(fmt.Sprintf("Key %s should have at least one record", app.Title))
		}
		last := allRevs[len(allRevs)-1]
		if last.DeleteTimeStamp == nil {
			return fmt.Errorf("Err: to be created application %s has been created", app.Title)
		}
		lastVersion, _ := strconv.Atoi(last.ResourceVersion)
		clone.ResourceVersion = strconv.Itoa(lastVersion + 1)
	} else {
		clone.ResourceVersion = strconv.Itoa(0)
	}
	clone.CreateTimeStamp = time.Now()
	s.db[clone.Title] = append(s.db[clone.Title], clone)
	return nil

}

// Update adds the new version of the key to the db
func (s *Store) Update(app *api.Application) error {
	s.Lock()
	defer s.Unlock()

	if err := validateApplication(app); err != nil {
		return err
	}
	allRevs, ok := s.db[app.Title]
	if !ok {
		return fmt.Errorf("Err: to be updated application %s has not been created", app.Title)
	}
	if len(allRevs) == 0 {
		panic(fmt.Sprintf("Key %s should have at least one record", app.Title))
	}
	last := allRevs[len(allRevs)-1]
	if last.DeleteTimeStamp != nil {
		return fmt.Errorf("Err: to be updated application %s has been deleted", app.Title)
	}
	if api.Equal(app, last) {
		return fmt.Errorf("Err: application %s is up-to-date, abort update", app.Title)
	}

	clone := api.DeepCopy(app)
	clone.CreateTimeStamp = last.CreateTimeStamp
	lastVersion, _ := strconv.Atoi(last.ResourceVersion)
	clone.ResourceVersion = strconv.Itoa(lastVersion + 1)
	s.db[clone.Title] = append(s.db[clone.Title], clone)
	return nil
}

// Delete adds a new version of the key with DeleteTimeStamp being set
func (s *Store) Delete(key string) error {
	s.Lock()
	defer s.Unlock()

	if allRevs, ok := s.db[key]; ok {
		if len(allRevs) == 0 {
			panic(fmt.Sprintf("Key %s should have at least one record", key))
		}

		last := allRevs[len(allRevs)-1]
		if last.DeleteTimeStamp != nil {
			return fmt.Errorf("Err: to be deleted application %s has been deleted", key)
		}
		clone := api.DeepCopy(last)
		now := &time.Time{}
		*now = time.Now()
		clone.DeleteTimeStamp = now
		lastVersion, _ := strconv.Atoi(last.ResourceVersion)
		clone.ResourceVersion = strconv.Itoa(lastVersion + 1)
		s.db[clone.Title] = append(s.db[clone.Title], clone)
		return nil
	}
	return fmt.Errorf("Err: application with title %s does not exist in the DB", key)
}

func inArray(input string, array []string) bool {
	for _, each := range array {
		if each == input {
			return true
		}
	}
	return false
}

func GetApplications(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/x-yaml")
	query := req.URL.Query()
	for k, _ := range query {
		if _, ok := supportedQueries[k]; !ok {
			http.Error(res, fmt.Sprintf("Err: the query key %s is not supported", k), http.StatusBadRequest)
			return
		}
	}
	ret, _ := appDB.List(query)
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fmt.Sprintf("%d applications are found", len(ret))))
	err := yaml.NewEncoder(res).Encode(ret)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func GetApplication(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/x-yaml")
	params := mux.Vars(req)
	t := params["title"]

	query := req.URL.Query()
	for k, _ := range query {
		if k != "dump" {
			http.Error(res, fmt.Sprintf("Err: query key %s is not supported, only \"dump\" is supported", k), http.StatusBadRequest)
			return
		}
	}
	_, dump := query["dump"]
	if !dump {
		ret, err := appDB.Get(t)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		res.WriteHeader(http.StatusOK)
		if err := yaml.NewEncoder(res).Encode(ret); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	} else {
		ret, err := appDB.DumpKey(t)
		if err != nil {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		res.WriteHeader(http.StatusOK)
		if err := yaml.NewEncoder(res).Encode(ret); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}
}

// To be extanded with more checks
func validateApplication(in *api.Application) error {
	if in.Title == "" {
		return fmt.Errorf("Err: application title cannot be empty")
	}
	if in.Version == "" {
		return fmt.Errorf("Err: application %s's version cannot be empty", in.Title)
	}
	var err error
	for _, each := range in.Maintainers {
		_, e := mail.ParseAddress(each.Email)
		if e != nil {
			err = fmt.Errorf("Err: application %s's maintainer has wrong email %s: %v", in.Title, each.Email, e)
			break
		}
	}
	return err
}

func reportResultsInResponse(res http.ResponseWriter, accepted []*api.Application, errs []error) {
	res.WriteHeader(http.StatusOK)
	var report string

	if len(errs) != 0 {
		for _, each := range errs {
			report = fmt.Sprintf("%s%v\n", report, each)
		}
	}
	if len(accepted) != 0 {
		for _, each := range accepted {
			report = fmt.Sprintf("%s%v has been created/updated successfully\n", report, each.Title)
		}
	}
	res.Write([]byte(report))
}

func forEachApplication(dc *yaml.Decoder, f func(*api.Application) error) ([]*api.Application, []error) {
	accepted := make([]*api.Application, 0)
	allErrs := make([]error, 0)

	for {
		var app api.Application
		err := dc.Decode(&app)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			allErrs = append(allErrs, err)
			break
		}
		err = f(&app)
		if err != nil {
			allErrs = append(allErrs, err)
		} else {
			accepted = append(accepted, &app)
		}
	}
	return accepted, allErrs
}

func CreateApplications(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/x-yaml" {
		http.Error(res, "Content-Type header is not text/x-yaml", http.StatusUnsupportedMediaType)
		return
	}
	res.Header().Set("Content-Type", "text/x-yaml")
	dc := yaml.NewDecoder(req.Body)

	created, allErrs := forEachApplication(dc, func(app *api.Application) error {
		return appDB.Create(app)
	})
	reportResultsInResponse(res, created, allErrs)
}

func UpdateApplications(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/x-yaml" {
		http.Error(res, "Content-Type header is not text/x-yaml", http.StatusUnsupportedMediaType)
		return
	}

	res.Header().Set("Content-Type", "text/x-yaml")
	dc := yaml.NewDecoder(req.Body)

	updated, allErrs := forEachApplication(dc, func(app *api.Application) error {
		return appDB.Update(app)
	})
	reportResultsInResponse(res, updated, allErrs)
}

func DeleteApplication(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/x-yaml")
	params := mux.Vars(req)
	t := params["title"]

	err := appDB.Delete(t)
	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fmt.Sprintf("%s is deleted successfully", t)))
}

var port string = "8082"

func SetFlags(serverPort string) {
	port = serverPort
}

// limited to 1 request per second, burst is 3. rateLimit is set to avoid bloating the memory in a short time.
var limiter = rate.NewLimiter(1, 3)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !limiter.Allow() {
			http.Error(res, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		log.Printf("%s %s%s %s", req.Method, req.Host, req.URL, req.Proto)
		next.ServeHTTP(res, req)
	})

}

func Init() {
	InitInMemDB()

	router.Use(RateLimiter)

	router.HandleFunc("/api/applications", GetApplications).Methods("GET")
	router.HandleFunc("/api/applications/{title}", GetApplication).Methods("GET")
	router.HandleFunc("/api/applications", CreateApplications).Methods("POST")
	router.HandleFunc("/api/applications", UpdateApplications).Methods("PUT")
	router.HandleFunc("/api/applications/{title}", DeleteApplication).Methods("DELETE")
}

func StartServer() {
	Init()
	log.Println("Server is listening on port " + port)
	server := http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	shutdownHandler := make(chan os.Signal, 2)
	stop := make(chan struct{})
	signal.Notify(shutdownHandler, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdownHandler
		close(stop)
		<-shutdownHandler
		os.Exit(1) // second signal. Exit directly.
	}()

	tick := time.NewTicker(DefaultCompactionInterval)
	go Compaction(tick, stop)

	log.Fatal(server.ListenAndServe())
}
