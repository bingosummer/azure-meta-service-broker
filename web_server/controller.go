package web_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bingosummer/azure-meta-service-broker/model"
	"github.com/bingosummer/azure-meta-service-broker/utils"
)

const (
	X_BROKER_API_VERSION_NAME = "X-Broker-Api-Version"
	X_BROKER_API_VERSION      = "2.5"
)

type Controller struct {
	serviceModules map[string]string
	catalog        model.Catalog
}

func NewController() *Controller {
	serviceModules := make(map[string]string)
	var catalog model.Catalog
	serviceModules, catalog, _ = loadServiceModulesAndCatalogs()

	ticker := time.NewTicker(time.Second * 30)
	go func() {
		for _ = range ticker.C {
			serviceModules, catalog, _ = loadServiceModulesAndCatalogs()
		}
	}()

	return &Controller{
		serviceModules: serviceModules,
		catalog:        catalog,
	}
}

func (c *Controller) Catalog(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Broker Catalog...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	apiVersion := r.Header.Get(X_BROKER_API_VERSION_NAME)
	supported := validateApiVersion(apiVersion, X_BROKER_API_VERSION)
	if !supported {
		fmt.Printf("API Version is %s, not supported.\n", apiVersion)
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}
	fmt.Println("API Version is " + apiVersion)

	utils.WriteResponse(w, http.StatusOK, c.catalog)
}

func (c *Controller) Provision(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Create Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	acceptsIncomplete := r.URL.Query().Get("accepts_incomplete")
	if acceptsIncomplete != "true" {
		fmt.Println("Only asynchronous provisioning is supported")
		response := make(map[string]string)
		response["error"] = "AsyncRequired"
		response["description"] = "This service plan requires client support for asynchronous service operations."
		utils.WriteResponse(w, 422, response)
		return
	}

	var instance model.ServiceInstance

	instanceId := utils.ExtractVarsFromRequest(r, "instance_id")
	instance.Id = instanceId

	err = utils.ProvisionDataFromRequest(r, &instance)
	if err != nil {
		fmt.Println("Failed to provision data from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	serviceId := instance.ServiceId
	serviceModulePath := utils.GetPath([]string{"service_modules", c.serviceModules[serviceId]})
	serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
	fmt.Println(serviceModuleExecutable)

	bytes, _ := json.Marshal(instance)
	args := []string{"-operation", "Provision", "-parameters", string(bytes)}
	fmt.Println(args)
	utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

	response := model.CreateServiceInstanceResponse{
		DashboardUrl: "",
	}
	utils.WriteResponse(w, http.StatusAccepted, response)
}

func (c *Controller) Poll(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Instance State....")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var instance model.ServiceInstance

	instanceId := utils.ExtractVarsFromRequest(r, "instance_id")
	instance.Id = instanceId

	serviceId := "2e2fc314-37b6-4587-8127-8f9ee8b33fea"
	serviceModulePath := utils.GetPath([]string{"service_modules", c.serviceModules[serviceId]})
	serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
	fmt.Println(serviceModuleExecutable)

	bytes, _ := json.Marshal(instance)
	args := []string{"-operation", "Poll", "-parameters", string(bytes)}
	fmt.Println(args)
	lastOperateionResponse := utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

	var response model.CreateLastOperationResponse

	if err := json.Unmarshal(lastOperateionResponse, &response); err != nil {
		panic(err)
	}

	if response.State == "Gone" {
		w.WriteHeader(http.StatusGone)
		return
	}

	utils.WriteResponse(w, http.StatusOK, response)
}

func (c *Controller) Deprovision(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Remove Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var instance model.ServiceInstance

	instanceId := utils.ExtractVarsFromRequest(r, "instance_id")
	instance.Id = instanceId

	serviceId := r.URL.Query().Get("service_id")
	serviceModulePath := utils.GetPath([]string{"service_modules", c.serviceModules[serviceId]})
	serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
	fmt.Println(serviceModuleExecutable)

	bytes, _ := json.Marshal(instance)
	args := []string{"-operation", "Deprovision", "-parameters", string(bytes)}
	fmt.Println(args)
	utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

	response := make(map[string]string)
	utils.WriteResponse(w, http.StatusOK, response)
}

func (c *Controller) Bind(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Bind Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var instance model.ServiceInstance

	instanceId := utils.ExtractVarsFromRequest(r, "instance_id")
	instance.Id = instanceId

	err = utils.ProvisionDataFromRequest(r, &instance)
	if err != nil {
		fmt.Println("Failed to provision data from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	serviceId := instance.ServiceId
	serviceModulePath := utils.GetPath([]string{"service_modules", c.serviceModules[serviceId]})
	serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
	fmt.Println(serviceModuleExecutable)

	bytes, _ := json.Marshal(instance)
	args := []string{"-operation", "Bind", "-parameters", string(bytes)}
	fmt.Println(args)
	credentialsBytes := utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

	var credentials model.Credentials

	if err := json.Unmarshal(credentialsBytes, &credentials); err != nil {
		panic(err)
	}

	response := model.CreateServiceBindingResponse{
		Credentials: credentials,
	}

	utils.WriteResponse(w, http.StatusCreated, response)
}

func (c *Controller) UnBind(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Unbind Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var instance model.ServiceInstance

	instanceId := utils.ExtractVarsFromRequest(r, "instance_id")
	instance.Id = instanceId

	serviceId := r.URL.Query().Get("service_id")
	serviceModulePath := utils.GetPath([]string{"service_modules", c.serviceModules[serviceId]})
	serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
	fmt.Println(serviceModuleExecutable)

	bytes, _ := json.Marshal(instance)
	args := []string{"-operation", "Unbind", "-parameters", string(bytes)}
	fmt.Println(args)
	utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

	response := make(map[string]string)
	utils.WriteResponse(w, http.StatusOK, response)
}

func authentication(r *http.Request) (int, error) {
	authUsername, authPassword, err := loadAuthCredentials()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return http.StatusUnauthorized, errors.New("No username and password provided in the request's Authorization header")
	}

	if username != authUsername || password != authPassword {
		return http.StatusUnauthorized, errors.New("The username and password are invalid")
	}

	fmt.Println("Passed authentication.")
	return 0, nil
}

func loadAuthCredentials() (string, string, error) {
	username := os.Getenv("authUsername")
	if username == "" {
		return "", "", errors.New("No auth_username provided in environment variables")
	}

	password := os.Getenv("authPassword")
	if password == "" {
		return "", "", errors.New("No auth_password provided in environment variables")
	}

	return username, password, nil
}

func validateApiVersion(actual, expected string) bool {
	apiVersion := strings.Split(actual, ".")
	majorApiVersionActual, err1 := strconv.Atoi(apiVersion[0])
	minorApiVersionActual, err2 := strconv.Atoi(apiVersion[1])
	if err1 != nil || err2 != nil {
		return false
	}

	apiVersion = strings.Split(expected, ".")
	majorApiVersionExpected, _ := strconv.Atoi(apiVersion[0])
	minorApiVersionExpected, _ := strconv.Atoi(apiVersion[1])

	if majorApiVersionActual < majorApiVersionExpected {
		return false
	}
	if majorApiVersionActual == majorApiVersionExpected && minorApiVersionActual < minorApiVersionExpected {
		return false
	}
	return true
}

func loadServiceModulesAndCatalogs() (map[string]string, model.Catalog, error) {
	serviceModules := make(map[string]string)
	var catalog model.Catalog

	serviceModulesPath := "service_modules"

	files, _ := ioutil.ReadDir(serviceModulesPath)
	for _, f := range files {
		serviceModuleName := f.Name()
		serviceModulePath := utils.GetPath([]string{serviceModulesPath, serviceModuleName})
		serviceModuleExecutable := "." + string(os.PathSeparator) + "main"
		args := []string{"-operation", "Catalog"}
		catalogBytes := utils.ExecCommand(serviceModuleExecutable, args, serviceModulePath)

		var serviceModuleCatalog model.Catalog
		if err := json.Unmarshal(catalogBytes, &serviceModuleCatalog); err != nil {
			return serviceModules, catalog, err
		}

		for _, service := range serviceModuleCatalog.Services {
			catalog.Services = append(catalog.Services, service)
			serviceModules[service.Id] = serviceModuleName
			fmt.Println("Loading the service module " + serviceModuleName + "(" + service.Id + "), catalog:\n")
			fmt.Println(string(catalogBytes))
		}
	}

	return serviceModules, catalog, nil
}
