/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package router

import (
	"goji.io"
	"goji.io/pat"

	"github.com/goxforum/xforum/pkg/controller"
	"github.com/goxforum/xforum/pkg/system"
	//"github.com/gorilla/csrf"
)

func NewRouter(app *system.Application) *goji.Mux {
	sp := goji.SubMux()
	h := controller.BaseHandler{App: app}

	sp.HandleFunc(pat.Get("/"), h.ArticleHomeList)
	sp.HandleFunc(pat.Get("/view"), h.ViewAtTpl)
	sp.HandleFunc(pat.Get("/feed"), h.FeedHandler)
	sp.HandleFunc(pat.Get("/robots.txt"), h.Robots)

	sp.HandleFunc(pat.Get("/n/:cid"), h.CategoryDetail)
	sp.HandleFunc(pat.Get("/member/:uid"), h.UserDetail)
	sp.HandleFunc(pat.Get("/tag/:tag"), h.TagDetail)
	sp.HandleFunc(pat.Get("/search"), h.SearchDetail)

	sp.HandleFunc(pat.Get("/logout"), h.UserLogout)
	sp.HandleFunc(pat.Get("/notification"), h.UserNotification)

	sp.HandleFunc(pat.Get("/t/:aid"), h.ArticleDetail)
	sp.HandleFunc(pat.Post("/t/:aid"), h.ArticleDetailPost)

	sp.HandleFunc(pat.Get("/setting"), h.UserSetting)
	sp.HandleFunc(pat.Post("/setting"), h.UserSettingPost)

	sp.HandleFunc(pat.Get("/newpost/:cid"), h.ArticleAdd)
	sp.HandleFunc(pat.Post("/newpost/:cid"), h.ArticleAddPost)

	sp.HandleFunc(pat.Get("/login"), h.UserLogin)
	sp.HandleFunc(pat.Post("/login"), h.UserLoginPost)
	sp.HandleFunc(pat.Get("/register"), h.UserLogin)
	sp.HandleFunc(pat.Post("/register"), h.UserLoginPost)

	sp.HandleFunc(pat.Get("/qqlogin"), h.QQOauthHandler)
	sp.HandleFunc(pat.Get("/oauth/qq/callback"), h.QQOauthCallback)
	sp.HandleFunc(pat.Get("/wblogin"), h.WeiboOauthHandler)
	sp.HandleFunc(pat.Get("/oauth/wb/callback"), h.WeiboOauthCallback)

	sp.HandleFunc(pat.Post("/content/preview"), h.ContentPreviewPost)
	sp.HandleFunc(pat.Post("/file/upload"), h.FileUpload)

	sp.HandleFunc(pat.Get("/admin/post/edit/:aid"), h.ArticleEdit)
	sp.HandleFunc(pat.Post("/admin/post/edit/:aid"), h.ArticleEditPost)
	sp.HandleFunc(pat.Get("/admin/comment/edit/:aid/:cid"), h.CommentEdit)
	sp.HandleFunc(pat.Post("/admin/comment/edit/:aid/:cid"), h.CommentEditPost)
	sp.HandleFunc(pat.Get("/admin/user/edit/:uid"), h.UserEdit)
	sp.HandleFunc(pat.Post("/admin/user/edit/:uid"), h.UserEditPost)
	sp.HandleFunc(pat.Get("/admin/user/list"), h.AdminUserList)
	sp.HandleFunc(pat.Post("/admin/user/list"), h.AdminUserListPost)
	sp.HandleFunc(pat.Get("/admin/category/list"), h.AdminCategoryList)
	sp.HandleFunc(pat.Post("/admin/category/list"), h.AdminCategoryListPost)
	sp.HandleFunc(pat.Get("/admin/link/list"), h.AdminLinkList)
	sp.HandleFunc(pat.Post("/admin/link/list"), h.AdminLinkListPost)

	return sp
}
