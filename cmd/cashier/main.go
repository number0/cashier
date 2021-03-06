package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"time"

	"github.com/nsheridan/cashier/client"
	"github.com/pkg/browser"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/agent"
)

var (
	u, _     = user.Current()
	cfg      = pflag.String("config", path.Join(u.HomeDir, ".cashier.conf"), "Path to config file")
	ca       = pflag.String("ca", "http://localhost:10000", "CA server")
	keysize  = pflag.Int("key_size", 2048, "Key size. Ignored for ed25519 keys")
	validity = pflag.Duration("validity", time.Hour*24, "Key validity")
	keytype  = pflag.String("key_type", "rsa", "Type of private key to generate - rsa, ecdsa or ed25519")
)

func main() {
	pflag.Parse()

	c, err := client.ReadConfig(*cfg)
	if err != nil {
		log.Fatalf("Error parsing config file: %v\n", err)
	}
	fmt.Printf("Your browser has been opened to visit %s\n", c.CA)
	if err := browser.OpenURL(c.CA); err != nil {
		fmt.Println("Error launching web browser. Go to the link in your web browser")
	}
	fmt.Println("Generating new key pair")
	priv, pub, err := client.GenerateKey(c.Keytype, c.Keysize)
	if err != nil {
		log.Fatalln("Error generating key pair: ", err)
	}

	fmt.Print("Enter token: ")
	var token string
	fmt.Scanln(&token)

	cert, err := client.Sign(pub, token, c)
	if err != nil {
		log.Fatalln(err)
	}
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatalln("Error connecting to agent: %s", err)
	}
	defer sock.Close()
	a := agent.NewClient(sock)
	if err := client.InstallCert(a, cert, priv); err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Credentials added.")
}
