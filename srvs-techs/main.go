package main

import (
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

var users = []*User{
	{Surname: "Medvedev", FirstName: "Aleksei", Patronym: "Olegovich", Position: "chief_expert"},
	{Surname: "Mangushev", FirstName: "Fedor", Patronym: "Pavlovich", Position: "department_head"},
	{Surname: "Mayorov", FirstName: "Aleksandr", Patronym: "Vladimirovich", Position: "chief_expert"},
	{Surname: "Lachugina", FirstName: "Natalia", Patronym: "Vadimovna", Position: "department_head"},
	{Surname: "Zhigalkin", FirstName: "Sergei", Patronym: "Aleksandrovich", Position: "town_mayor"},
	{Surname: "Polevshchikov", FirstName: "Sergei", Patronym: "Petrovich", Position: "mayors_first_deputy"},
}

var techs = []*TechElem{
	{InvNumber: 1400001, DevType: "system_unit", Department: "mayors_ofice", User: *users[4], Status: "in_use"},
	{InvNumber: 1400025, DevType: "system_unit", Status: "decommissioned"},
	{InvNumber: 1400184, DevType: "system_unit", Department: "IT_department", User: *users[0], Status: "in_use"},
	{InvNumber: 1400399, DevType: "printing_device", Department: "road_construction_department", Status: "in_use"},
	{InvNumber: 1400398, DevType: "printing_device", Status: "decommissioned"},
	{InvNumber: 1400186, DevType: "uninterruptable_power_source", Department: "IT_department", User: *users[2], Status: "in_use"},
	{InvNumber: 1400695, DevType: "projector", Department: "conference_hall", Status: "in_use"},
}

func GetTechs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" {
		filteredTechs := []TechElem{}
		for _, item := range techs {
			if item.Status == status {
				filteredTechs = append(filteredTechs, *item)
			}
		}
		json.NewEncoder(w).Encode(filteredTechs)
	} else {
		json.NewEncoder(w).Encode(techs)
	}
}

func UpdateTechStatus(w http.ResponseWriter, r *http.Request) {
	inventarnik, _ := strconv.Atoi(chi.URLParam(r, "invnumber"))

	var newStatus struct {
		Status string `json:"status"`
	}

	err := json.NewDecoder(r.Body).Decode(&newStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i, item := range techs {
		if inventarnik == item.InvNumber {
			techs[i].Status = newStatus.Status
			json.NewEncoder(w).Encode(techs[i])
			return
		}
	}

	http.NotFound(w, r)
}

func main() {

	logger := httplog.NewLogger("service-techs", httplog.Options{
		JSON: true,
	})

	r := chi.NewRouter()
	r.Use(chiprometheus.NewMiddleware("service-techs"))
	r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/techs", GetTechs)
	r.Post("/techs/{invnumber}", UpdateTechStatus)

	http.ListenAndServe(":8080", r)
}
