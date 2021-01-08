package auth

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sparrow/etcdv3"

	"github.com/coreos/etcd/clientv3"
	"golang.org/x/crypto/bcrypt"
)

var Userbase = "user"
var swPath = "sw"
var inited = false

type UserInfo struct {
	Name        string `json:"name"`
	Tel         string `json:"tel"`
	Email       string `json:"email"`
	Permissions string `json:"permissions"`
}

type UserStoreInfo struct {
	Password string    `json:"sw"`
	Info     *UserInfo `json:"info"`
}

func newUser(info *UserStoreInfo) error {
	cli, err := etcdv3.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	path := filepath.Join(*(etcdv3.Basepath), Userbase, info.Info.Name)
	presp, err := cli.Get(context.Background(), path, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	if presp.Count > 0 {
		return errors.New("user existed!")
	}
	value, err := json.Marshal(info.Info)
	if err != nil {
		return nil
	}
	_, err = cli.Put(context.Background(), path, string(value))
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(info.Password), bcrypt.DefaultCost) //加密处理
	if err != nil {
		return err
	}

	_, err = cli.Put(context.Background(), filepath.Join(path, swPath), string(hash))
	if err != nil {
		cli.Delete(context.Background(), path, clientv3.WithPrefix())
		return err
	}
	return nil
}

func delUser(name string) error {
	cli, err := etcdv3.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	key := filepath.Join(*(etcdv3.Basepath), Userbase, name)

	if _, err := cli.Delete(context.Background(), key, clientv3.WithPrefix()); err != nil {
		return err
	}
	return nil
}

func verifyUser(name, password string) error {
	if initAuth() != nil {
		return errors.New("have not inited")
	}
	cli, err := etcdv3.NewClient()
	if err != nil {
		return errors.New("internal error")
	}
	defer cli.Close()
	path := filepath.Join(*etcdv3.Basepath, Userbase, name, swPath)
	presp, err := cli.Get(context.Background(), path)
	if err != nil {
		return errors.New("internal error")
	}
	if presp.Count == 0 {
		return errors.New("user or pwd is not correct")
	}

	err = bcrypt.CompareHashAndPassword(presp.Kvs[0].Value, []byte(password))
	if err != nil {
		return errors.New("user or pwd is not correct")
	}
	return nil
}

func changePw(name, oldPw, newPw string) error {
	err := verifyUser(name, oldPw)
	if err != nil {
		return err
	}

	cli, err := etcdv3.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(newPw), bcrypt.DefaultCost) //加密处理
	if err != nil {
		return errors.New("internal error")
	}

	path := filepath.Join(*(etcdv3.Basepath), Userbase, name, swPath)
	_, err = cli.Put(context.Background(), path, string(hash))
	if err != nil {
		return err
	}
	return nil
}

func updateInfo(info *UserInfo) error {
	cli, err := etcdv3.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	path := filepath.Join(*(etcdv3.Basepath), Userbase, info.Name)
	presp, err := cli.Get(context.Background(), path)
	if err != nil || presp.Count == 0 {
		return errors.New("internal error")
	}
	origin := UserInfo{}
	err = json.Unmarshal([]byte(presp.Kvs[0].Value), &origin)
	if err != nil {
		return errors.New("internal error")
	}
	if len(info.Permissions) > 0 {
		origin.Permissions = info.Permissions
	}
	if len(info.Email) > 0 {
		origin.Email = info.Email
	}
	if len(info.Tel) > 0 {
		origin.Tel = info.Tel
	}

	v, err := json.Marshal(origin)
	if err != nil {
		return errors.New("internal error")
	}
	_, err = cli.Put(context.Background(), path, string(v))
	if err != nil {
		return errors.New("internal error")
	}
	return nil
}

func initAuth() error {
	if inited {
		return nil
	}
	cli, err := etcdv3.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	key := filepath.Join(*(etcdv3.Basepath), Userbase, "admin", swPath)
	presp, err := cli.Get(context.Background(), key)
	if err != nil {
		return err
	}
	if presp.Count > 0 {
		inited = true
		return nil
	}

	info := UserStoreInfo{
		Password: "admin",
		Info: &UserInfo{
			Name:        "admin",
			Tel:         "13812345678",
			Email:       "admin@example.com",
			Permissions: "/",
		},
	}

	if newUser(&info) != nil {
		return errors.New("init failed")
	}

	inited = true
	return nil
}

func getUserInfo(name string) (*UserInfo, error) {
	cli, err := etcdv3.NewClient()
	if err != nil {
		return nil, errors.New("internal error")
	}
	defer cli.Close()
	key := filepath.Join(*etcdv3.Basepath, Userbase, name)
	presp, err := cli.Get(context.Background(), key)
	if err != nil || presp.Count == 0 {
		return nil, errors.New("internal error")
	}
	userInfo := UserInfo{}
	err = json.Unmarshal([]byte(presp.Kvs[0].Value), &userInfo)
	if err != nil {
		return nil, errors.New("internal error")
	}
	return &userInfo, nil

}
