package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/idcooldi/telegram-bot-api"
)

const (
	//Password type mode autorization on enter password
	Password   = 1
	//Key type mode autorization on public_key
	Key        = 2
	//DefTimeout timeout by default
	DefTimeout = 3 // second
)

//Config структура для Json "Config"
type config struct {
	SSH struct {
		User  string `json:"user"`
		Host  string `json:"host"`
		Port  int    `json:"port"`
		Cert  string `json:"cert"`
		Token string `json:"token"`
		Proxy string `json:"proxy"`
		Mode int `json:"mode"`
	} `json:"ssh"`
}
//SSH struct for config
type shell struct {
	IP      string
	User    string
	Cert    string //password or key file path
	Port    int
	session *ssh.Session
	client  *ssh.Client
}

var conf config

//Conf func for download file wich config
func readConfing() {

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	fuck("Error pwd:", err)

	configFile, err := ioutil.ReadFile(dir + "/config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configFile, &conf)
	fuck("Ошибка Unmarshal", err)

}
func (sshClient *shell) readPublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
//Connect - create connecting session of ssh
func (sshClient *shell) connect(mode int) {

	var sshConfig *ssh.ClientConfig
	var auth []ssh.AuthMethod
	if mode == Password {
		auth = []ssh.AuthMethod{ssh.Password(sshClient.Cert)}
	} else if mode == Key {
		auth = []ssh.AuthMethod{sshClient.readPublicKeyFile(sshClient.Cert)}
	} else {
		log.Println("does not support mode: ", mode)
		return
	}

	sshConfig = &ssh.ClientConfig{
		User: sshClient.User,
		Auth: auth,

		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: time.Second * DefTimeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshClient.IP, sshClient.Port), sshConfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	session, err := client.NewSession()
	if err != nil {
		log.Printf("Error of NewSession %v\n",err)
		err:=client.Close()
		fuck("Error of close client",err)
		return
	}

	sshClient.session = session
	sshClient.client = client
}
//RunCmd do command line
func (sshClient *shell) RunCmd(cmd string) string {
	sshClient.connect(conf.SSH.Mode)
	out, err := sshClient.session.CombinedOutput(cmd)
	fuck("Error of RunCmd", err)
	sshClient.Close()
	return string(out)
}
//Close close session of ssh
func (sshClient *shell) Close() {
	err := sshClient.session.Close()
	fuck("Error of close session",err)
	err = sshClient.client.Close()
	fuck("Error of close client",err)
}

//demo
func main() {
	readConfing()
	client := &shell{
		IP:   conf.SSH.Host,
		User: conf.SSH.User,
		Port: conf.SSH.Port,
		Cert: conf.SSH.Cert,
	}
	bot, err := tgbotapi.NewBotAPI(conf.SSH.Token, "socks5", conf.SSH.Proxy, nil)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	fuck("Error of GetUpdatesChan",err)

	//client.Connect(Key)
	for update := range updates {
		if update.Message == nil {
			continue
		}
		s := strings.TrimPrefix(update.Message.Text, "/")
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			ms := client.RunCmd(s)
			msg.Text = ms
			_,err:=bot.Send(msg)
			fuck("Error of send Bot",err)

		}

	}
	//client.Close()
}
func fuck(s string,e error){
	if e != nil {
		log.Printf(s+" %v\n",e)
	}
}