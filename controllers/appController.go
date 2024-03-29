package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/l3njo/dropnote-web/models"
	uuid "github.com/satori/go.uuid"
)

var funcMap = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
}

// IndexHandler handles the "/", "/home", "/favicon.ico" and all undefined routes.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		http.ServeFile(w, r, "static/img/favicon.ico")
		return
	}

	session, err := store.Get(r, sessionCookie)
	Handle(err)
	isAuth := checkAuth(session)
	data := Page{
		Data: Data{
			Title:       "Home",
			Site:        site,
			Link:        r.URL.Path,
			Description: "DropNote Home",
		},
	}

	meta := filepath.Join("templates", "meta", "home.html.tmpl")
	body := filepath.Join("templates", "home.html.tmpl")
	if isAuth {
		uData := session.Values["data"].(*models.User)
		data.Name, data.Mail = uData.Name, uData.Mail
	}

	if r.URL.Path != "/" && r.URL.Path != "/home" {
		displayHTTPError(w, r, http.StatusNotFound)
		return
	}

	if informed, ok := session.Values["isInformed"].(bool); !ok || !informed {
		data.Flashes = append(data.Flashes, Flash{Message: "Have a cookie! This site uses cookies to personalize your experience.", Status: info})
		session.Values["isInformed"] = true
	}

	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, v := range flashes {
			data.Flashes = append(data.Flashes, *v.(*Flash))
		}
	}

	Handle(session.Save(r, w))
	tmpl, err := template.ParseFiles(base, meta, body)
	Handle(err)
	tmpl.ExecuteTemplate(w, "layout", data)
	return
}

// DropNoteHandler handles the "/dropnote" route.
func DropNoteHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, sessionCookie)
	Handle(err)
	isAuth := checkAuth(session)
	data := Page{
		Data: Data{
			Title:       "Drop Note",
			Site:        site,
			Link:        r.URL.Path,
			Description: "Drop a Note",
		},
	}
	meta := filepath.Join("templates", "meta", "drop.html.tmpl")
	body := filepath.Join("templates", "dropnote.html.tmpl")
	if isAuth {
		uData := session.Values["data"].(*models.User)
		data.Name, data.Mail = uData.Name, uData.Mail
	}

	if informed, ok := session.Values["isInformed"].(bool); !ok || !informed {
		data.Flashes = append(data.Flashes, Flash{Message: "Have a cookie! This site uses cookies to personalize your experience.", Status: info})
		session.Values["isInformed"] = true
	}

	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, v := range flashes {
			data.Flashes = append(data.Flashes, *v.(*Flash))
		}
	}

	if r.Method == "POST" {
		r.ParseForm()
		note := &models.Note{
			Subject: r.Form["subject"][0],
			Content: r.Form["content"][0],
		}

		auth := ""
		if isAuth && len(r.Form["shouldLink"]) > 0 {
			if shouldLink := r.Form["shouldLink"][0]; shouldLink == "on" {
				sessionData := session.Values["data"].(*models.User)
				auth = sessionData.Auth
			}
		}

		if err := note.Post(auth); err != nil {
			Handle(err)
			session.AddFlash(Flash{Message: "Save failed.", Status: warning})
			Handle(session.Save(r, w))
			displayHTTPError(w, r, http.StatusInternalServerError)
			return
		}
		Handle(note.ParseDate())
		data.Note = *note
		session.AddFlash(Flash{Message: "Note saved.", Status: success})
		Handle(session.Save(r, w))
		http.Redirect(w, r, fmt.Sprintf("/dropcode?voucher=%s", note.Voucher), http.StatusFound)
		return
	}

	Handle(session.Save(r, w))
	tmpl, err := template.ParseFiles(base, meta, body)
	Handle(err)
	tmpl.ExecuteTemplate(w, "layout", data)
	return
}

// DropCodeHandler handles the "/dropcode" route.
func DropCodeHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, sessionCookie)
	Handle(err)
	isAuth := checkAuth(session)
	uData := &models.User{}
	data := Page{
		Data: Data{
			Title:       "Drop Code",
			Site:        site,
			Link:        r.URL.Path,
			Description: "Drop a Code",
		},
	}

	meta := filepath.Join("templates", "meta", "drop.html.tmpl")
	body := filepath.Join("templates", "dropcode.html.tmpl")
	if isAuth {
		uData = session.Values["data"].(*models.User)
		data.Name, data.Mail = uData.Name, uData.Mail
	}

	if informed, ok := session.Values["isInformed"].(bool); !ok || !informed {
		data.Flashes = append(data.Flashes, Flash{Message: "Have a cookie! This site uses cookies to personalize your experience.", Status: info})
		session.Values["isInformed"] = true
	}

	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, v := range flashes {
			data.Flashes = append(data.Flashes, *v.(*Flash))
		}
	}

	if keys, ok := r.URL.Query()["voucher"]; ok && len(keys) > 0 {
		note := &models.Note{Voucher: keys[0]}
		if err := note.ValidateGet(); err != nil {
			session.AddFlash(Flash{Message: "Retrieval failed.", Status: warning})
			data.Heading, data.Message = "Error!", err.Error()
			meta = filepath.Join("templates", "meta", "info.html.tmpl")
			body = filepath.Join("templates", "info.html.tmpl")
		} else if err := note.Get(uData.Auth); err != nil {
			Handle(err)
			session.AddFlash(Flash{Message: "Retrieval failed.", Status: warning})
			Handle(session.Save(r, w))
			displayHTTPError(w, r, http.StatusInternalServerError)
			return
		} else if *note == (models.Note{}) {
			session.AddFlash(Flash{Message: "Retrieval failed.", Status: warning})
			data.Heading, data.Message = "Error!", "That note does not exist"
			meta = filepath.Join("templates", "meta", "info.html.tmpl")
			body = filepath.Join("templates", "info.html.tmpl")
		} else if uuid.Equal(uuid.FromStringOrNil(note.Voucher), uuid.Nil) {
			session.AddFlash(Flash{Message: "Retrieval failed.", Status: warning})
			data.Heading, data.Message = "Error!", "That note does not exist"
			meta = filepath.Join("templates", "meta", "info.html.tmpl")
			body = filepath.Join("templates", "info.html.tmpl")
		} else {
			Handle(note.ParseDate())
			data.Note = *note
			session.AddFlash(Flash{Message: "Note retrieved.", Status: success})
			meta = filepath.Join("templates", "meta", "note.html.tmpl")
			body = filepath.Join("templates", "note.html.tmpl")
		}
	}

	Handle(session.Save(r, w))
	tmpl, err := template.ParseFiles(base, meta, body)
	Handle(err)
	tmpl.ExecuteTemplate(w, "layout", data)
	return
}
