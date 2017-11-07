package controller

import (
	"net/http"
)

func (h *BaseHandler) ViewAtTpl(w http.ResponseWriter, r *http.Request) {
	tpl := r.FormValue("tpl")
	if tpl == "desktop" || tpl == "mobile" {
		h.SetCookie(w, "tpl", tpl, 1)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
