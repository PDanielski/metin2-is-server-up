package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"

	"gopkg.in/gomail.v2"
	"gopkg.in/yaml.v2"
)

//StatusNotifier role is constantly checking for server status
type StatusNotifier struct {
	Server Server
}

//Status checkes for a server status and pushes the result in the returned channel
func (s StatusNotifier) Status() <-chan bool {
	ch := make(chan bool)
	go func() {
		conn, err := net.DialTimeout("tcp", s.Server.ServerName(), time.Duration(s.Server.Timeout)*time.Second)
		if err != nil {
			fmt.Printf("%s off\n", s.Server.ServerName())
		} else {
			fmt.Printf("%s on\n", s.Server.ServerName())
		}
		if err != nil {
			ch <- false
		} else {
			conn.Close()
			ch <- true
		}
	}()
	return ch
}

func main() {
	//Parse che configuration
	yamlConfig, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Fatal(fmt.Errorf("Error while reading conf.yml: %v", err))
	}
	err = yaml.Unmarshal(yamlConfig, &Config)
	if err != nil {
		log.Fatal(fmt.Errorf("Error while parsing yaml: %v", err))
	}

	//Check the sender email
	fmt.Println("Checking sender email credentials...")
	d := createDialer()
	conn, err := d.Dial()
	if err != nil {
		log.Fatal(fmt.Errorf("Couldn't connect to smtp host: %+v", err))
	}
	conn.Close()

	fmt.Println("OK credentials")

	//Stores the map key -> server
	servers := Config.Servers

	notifiers := make(map[string]*StatusNotifier)
	for key, s := range servers {
		notifiers[key] = &StatusNotifier{s}
		s.Status = false
		servers[key] = s
	}

	//If this flag is true, then it means we must notify receivers about the state change
	statusChanged := false

	isFirstIteration := true

	//Holds the channel for delaying the checking every given time
	var after <-chan time.Time

	for {
		if after != nil {
			<-after
		}

		fmt.Println("Checking statuses...")

		chs := make(map[string](<-chan bool))

		for key, n := range notifiers {
			chs[key] = n.Status()
		}

		for key, ch := range chs {
			status := <-ch
			if status != servers[key].Status && !isFirstIteration {
				statusChanged = true
			}
			s := servers[key]
			s.Status = status
			servers[key] = s
		}

		if statusChanged {
			notifyStatusChange(servers)
		}

		statusChanged = false
		isFirstIteration = false
		after = time.After(time.Duration(Config.CheckRate) * time.Second)
	}

}

func createDialer() *gomail.Dialer {
	sender := Config.Email.Sender
	return gomail.NewDialer(sender.Auth.Host, sender.Auth.Port, sender.Auth.Username, sender.Auth.Password)
}

func notifyStatusChange(srvs map[string]Server) {
	raw := `
	<b>Hi,</b>the server state changed:<br/>
	<table>
		<thead>
			<tr>
				<th>Code</th>
				<th>Server</th>
				<th>Status</th>
			</tr>
			{{ range $key, $server := .}}
			<tr>
				<td>{{ $key }}</td>
				<td>{{ $server.ServerName }}</td>
				<td>
				{{ if $server.Status }}
					<span style='color:green;'>Online</span>
				{{ else }}
					<span style='color:red;'>Offline</span>
				{{ end}}
				</td>
			</tr>
			{{ end }}
		</thead>
	</table>`

	tmpl, _ := template.New("mail").Parse(raw)

	m := gomail.NewMessage()
	m.SetHeader("From", Config.Email.Sender.Addr)
	m.SetHeader("To", Config.Email.Receivers...)
	m.SetHeader("Subject", "The server state changed")
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return tmpl.Execute(w, srvs)
	})
	err := createDialer().DialAndSend(m)
	if err != nil {
		fmt.Printf("Coudn't send email from %s\n", Config.Email.Sender.Addr)
	}
}
