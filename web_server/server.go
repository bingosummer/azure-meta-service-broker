package web_server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type Server struct {
	controller *Controller
}

func NewServer() *Server {
	controller := NewController()
	if controller == nil {
		return nil
	}

	return &Server{controller: controller}
}

func (s *Server) Start(defaultPort string) {
	router := mux.NewRouter()

	router.HandleFunc("/v2/catalog", s.controller.Catalog).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", s.controller.Provision).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", s.controller.Poll).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", s.controller.Deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", s.controller.Bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", s.controller.UnBind).Methods("DELETE")

	http.Handle("/", router)

	port := os.Getenv("VCAP_APP_PORT")
	if port == "" {
		port = defaultPort
	}
	fmt.Println("Server started, listening on port " + port + "...")
	http.ListenAndServe(":"+port, nil)
}
