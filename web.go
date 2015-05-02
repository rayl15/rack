package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/nlogger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"

	"github.com/convox/kernel/controllers"
)

var port string = "5000"

func redirect(path string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, path, http.StatusFound)
	}
}

func parseForm(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// r.ParseMultipartForm(2048)
	next(rw, r)
}

func authRequired(rw http.ResponseWriter) {
	rw.Header().Set("WWW-Authenticate", `Basic realm="Convox"`)
	rw.WriteHeader(401)
	rw.Write([]byte("unauthorized"))
}

func basicAuthentication(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.RequestURI == "/check" {
		next(rw, r)
		return
	}

	if password := os.Getenv("HTTP_PASSWORD"); password != "" {
		auth := r.Header.Get("Authorization")

		if auth == "" {
			authRequired(rw)
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			authRequired(rw)
			return
		}

		c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))

		if err != nil {
			return
		}

		parts := strings.SplitN(string(c), ":", 2)

		if len(parts) != 2 || parts[1] != password {
			authRequired(rw)
			return
		}
	}

	next(rw, r)
}

func check(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok"))
}

func main() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	router := mux.NewRouter()

	router.HandleFunc("/", redirect("/apps")).Methods("GET")

	router.HandleFunc("/check", check).Methods("GET")

	router.HandleFunc("/apps", controllers.AppList).Methods("GET")
	router.HandleFunc("/apps", controllers.AppCreate).Methods("POST")
	router.HandleFunc("/apps/{app}", controllers.AppShow).Methods("GET")
	router.HandleFunc("/apps/{app}", controllers.AppDelete).Methods("DELETE")
	router.HandleFunc("/apps/{app}/builds", controllers.AppBuilds).Methods("GET")
	router.HandleFunc("/apps/{app}/build", controllers.BuildCreate).Methods("POST")
	router.HandleFunc("/apps/{app}/changes", controllers.AppChanges).Methods("GET")
	router.HandleFunc("/apps/{app}/logs", controllers.AppLogs)
	router.HandleFunc("/apps/{app}/logs/stream", controllers.AppStream)
	router.HandleFunc("/apps/{app}/processes/{process}", controllers.ProcessShow).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/logs", controllers.ProcessLogs).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/logs/stream", controllers.ProcessStream)
	router.HandleFunc("/apps/{app}/processes/{process}/resources", controllers.ProcessResources).Methods("GET")
	router.HandleFunc("/apps/{app}/promote", controllers.AppPromote).Methods("POST")
	router.HandleFunc("/apps/{app}/releases", controllers.AppReleases).Methods("GET")
	router.HandleFunc("/apps/{app}/resources", controllers.AppResources).Methods("GET")
	router.HandleFunc("/apps/{app}/services", controllers.AppServices).Methods("GET")
	router.HandleFunc("/apps/{app}/status", controllers.AppStatus).Methods("GET")

	router.HandleFunc("/settings", controllers.SettingsList).Methods("GET")
	router.HandleFunc("/settings", controllers.SettingsUpdate).Methods("POST")

	n := negroni.New(
		negroni.NewRecovery(),
		nlogger.New("ns=kernel", nil),
		negroni.NewStatic(http.Dir("public")),
	)

	n.Use(negroni.HandlerFunc(parseForm))
	n.Use(negroni.HandlerFunc(basicAuthentication))
	n.UseHandler(router)
	n.Run(fmt.Sprintf(":%s", port))
}