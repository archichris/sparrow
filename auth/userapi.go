package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"sparrow/etcdv3"

	"github.com/coreos/etcd/clientv3"
)

func init() {
	HandleFunc("/api/v1/user/self/permissions", GetPermissions)
	HandleFunc("/api/v1/user/all", GetAllUsers)
	HandleFunc("/api/v1/user", UserProcess)

	initAuth()
}

// Create the Signin handler
func Signin(w http.ResponseWriter, r *http.Request) {
	// Get the JSON body and decode into credentials
	username := r.FormValue("uname")
	password := r.FormValue("passwd")

	if err := verifyUser(username, password); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := SetToken(w, username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func GetPermissions(w http.ResponseWriter, r *http.Request) {
	username, err := CheckToken(w, r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}

	info, err := getUserInfo(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	io.WriteString(w, info.Permissions)
}

type UserInfoGrid struct {
	Total int        `json:"total"`
	Rows  []UserInfo `json:"rows"`
}

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	username, err := CheckToken(w, r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}

	if username != "admin" {
		w.WriteHeader(http.StatusUnauthorized)
	}

	cli, err := etcdv3.NewClient()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer cli.Close()
	path := filepath.Join(*etcdv3.Basepath, Userbase)
	presp, err := cli.Get(context.Background(), path, clientv3.WithPrefix())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if presp.Count == 0 {
		w.WriteHeader(http.StatusInternalServerError)
	}

	grid := UserInfoGrid{Total: 1, Rows: []UserInfo{}}

	for _, kv := range presp.Kvs {

		if filepath.Base(string(kv.Key)) == swPath {
			continue
		}

		info := UserInfo{}
		if nil != json.Unmarshal(kv.Value, &info) {
			continue
		}
		grid.Rows = append(grid.Rows, info)

	}

	v, err := json.Marshal(grid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	io.WriteString(w, string(v))

}

func UserProcess(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		DelUser(w, r)
	case http.MethodPost:
		AddUser(w, r)
	case http.MethodPut:
		UpdateUser(w, r)
	case http.MethodGet:
		GetUser(w, r)

	}
}

func DelUser(w http.ResponseWriter, r *http.Request) {
	user := filepath.Base(r.URL.EscapedPath())
	if user == "admin" {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "admin can not be deleted.")
	}
	if nil != delUser(user) {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	user := filepath.Base(r.URL.EscapedPath())
	pw := r.FormValue("password")
	email := r.FormValue("email")
	tel := r.FormValue("tel")
	permissions := r.FormValue("permissions")

	if nil != newUser(&UserStoreInfo{Password: pw,
		Info: &UserInfo{Name: user, Email: email, Tel: tel, Permissions: permissions}}) {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	user := filepath.Base(r.URL.EscapedPath())
	userInfo, err := getUserInfo(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	v, err := json.Marshal(&userInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(v))
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	user := filepath.Base(r.URL.EscapedPath())
	email := r.FormValue("email")
	tel := r.FormValue("tel")
	permissions := r.FormValue("permissions")

	if nil != updateInfo(&UserInfo{Name: user, Email: email, Tel: tel, Permissions: permissions}) {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
