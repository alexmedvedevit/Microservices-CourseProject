package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	chiprometheus "github.com/nathan-jones/chi-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type TechElem struct {
	InvNumber  int    `json:"invnumber"`
	DevType    string `json:"devtype"`
	Department string `json:"department"`
	User       User   `json:"user"`
	Status     string `json:"status"`
}

type User struct {
	Surname   string `json:"surname"`
	FirstName string `json:"firstname"`
	Patronym  string `json:"patronym"`
	Position  string `json:"position"`
}

type TechStatusRequest struct {
	Status string `json:"status"`
}

// Вывод всей списанной техники
func GetDecommissionedTechs() ([]TechElem, error) {
	resp, err := http.Get("http://host.docker.internal:8080/techs?status=decommissioned")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var response []TechElem
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Вывод всей работающей техники
func GetTechsInUse() ([]TechElem, error) {
	resp, err := http.Get("http://host.docker.internal:8080/techs?status=in_use")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var response []TechElem
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Списание работающей техники
func SendTechToDump(inventarnik int) (*TechElem, error) {
	decommissionRequest := TechStatusRequest{
		Status: "decommissioned",
	}

	jsonBytes, err := json.Marshal(decommissionRequest)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("http://host.docker.internal:8080/techs/"+strconv.Itoa(inventarnik), "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var item TechElem
	err = json.NewDecoder(resp.Body).Decode(&item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func main() {

	logger := httplog.NewLogger("service-reqs", httplog.Options{
		JSON: true,
	})

	r := chi.NewRouter()
	r.Use(chiprometheus.NewMiddleware("service-reqs"))
	r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/in_use", func(w http.ResponseWriter, r *http.Request) {
		items, err := GetTechsInUse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	})

	r.Get("/decommissioned", func(w http.ResponseWriter, r *http.Request) {
		items, err := GetDecommissionedTechs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	})

	r.Post("/send-to-dump/{invnumber}", func(w http.ResponseWriter, r *http.Request) {
		inventarnik, err := strconv.Atoi(chi.URLParam(r, "invnumber"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		item, err := SendTechToDump(inventarnik)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if item == nil {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(item)
	})

	http.ListenAndServe(":8081", r)
}
