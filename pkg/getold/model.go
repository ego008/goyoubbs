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

package getold

type OldArticle struct {
	Code         int    `json:"code"`
	Id           string `json:"id"`
	Uid          string `json:"uid"`
	Cid          string `json:"cid"`
	RUid         string `json:"ruid"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Tags         string `json:"tags"`
	AddTime      string `json:"addtime"`
	EditTime     string `json:"edittime"`
	Comments     string `json:"comments"`
	CloseComment string `json:"closecomment"`
	Visible      string `json:"visible"`
	Views        string `json:"views"`
}
type OldUser struct {
	Code          int    `json:"code"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	Flag          string `json:"flag"`
	Avatar        string `json:"avatar"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	Url           string `json:"url"`
	Articles      string `json:"articles"`
	Replies       string `json:"replies"`
	RegTime       string `json:"regtime"`
	LastPostTime  string `json:"lastposttime"`
	LastReplyTime string `json:"lastreplytime"`
	LastLoginTime string `json:"lastLoginTime"`
	About         string `json:"about"`
	Notice        string `json:"notic"`
}
type OldCategory struct {
	Code     int    `json:"code"`
	Id       string `json:"id"`
	Name     string `json:"name"`
	Articles string `json:"articles"`
	About    string `json:"about"`
	Comments string `json:"comments"`
	Hidden   bool   `json:"hidden"`
}
type OldComment struct {
	Code    int    `json:"code"`
	Id      string `json:"id"`
	Aid     string `json:"articleid"`
	Uid     string `json:"uid"`
	Content string `json:"content"`
	AddTime string `json:"addtime"`
}
type OldQQ struct {
	Code   int    `json:"code"`
	Id     string `json:"id"`
	Uid    string `json:"uid"`
	Name   string `json:"name"`
	Openid string `json:"openid"`
}
type OldWeibo struct {
	Code   int    `json:"code"`
	Id     string `json:"id"`
	Uid    string `json:"uid"`
	Name   string `json:"name"`
	Openid string `json:"openid"`
}
type OldTag struct {
	Code     int    `json:"code"`
	Id       string `json:"id"`
	Name     string `json:"name"`
	Articles string `json:"articles"`
	Ids      string `json:"ids"`
}
