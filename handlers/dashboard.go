package handlers

import (
	"html/template"
	"net/http"

	appmiddleware "accountingweb/middleware"
)

var dashboardTmpl = template.Must(template.ParseFiles("templates/dashboard.html"))

type dashboardPageData struct {
	Username string
}

func DashboardGet(w http.ResponseWriter, r *http.Request) {
	username := "Stranger"

	// Since the view is unprotected, we cannot use `appmiddleware.GetUsername(r)`,
	// as it requires authentication already run for that request. 
	// Therefore, we skip middleware and directly access the session's value for username
	session, err := appmiddleware.GetSession(r)
	if err == nil {
		if u, ok := session.Values[appmiddleware.SessionKeyUsername].(string); ok && u != "" {
			username = u
		}
	}

	dashboardTmpl.Execute(w, dashboardPageData{
		Username: username,
	})
}
