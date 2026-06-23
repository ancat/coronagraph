package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"coronagraph/proxy"
	"coronagraph/service"
	"coronagraph/vault"
)

var (
	servePort     int
	serveCACert   string
	serveCAKey    string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the MITM proxy",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVar(&servePort, "port", 11111, "port to listen on")
	serveCmd.Flags().StringVar(&serveCACert, "ca-cert", "rootCA.pem", "CA certificate PEM file")
	serveCmd.Flags().StringVar(&serveCAKey, "ca-cert-key", "rootCA-key.pem", "CA private key PEM file")
}

func get_secrets() map[string]string {
	var local_vault vault.LocalVault

	if err := local_vault.LoadFromFile("config-lv.yml"); err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "Passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	defer clear(passphrase)

	contents, err := local_vault.ReadData(passphrase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decrypt vault -- incorrect passphrase? Error: %s\n", err)
		panic("")
	}

	secret_env, err := godotenv.Unmarshal(string(contents))
	if err != nil {
		panic(err)
	}

	return secret_env
}


func runServe(cmd *cobra.Command, args []string) error {
	if servePort < 1 || servePort > 65535 {
		return fmt.Errorf("invalid port: %d", servePort)
	}

	ca, err := proxy.LoadCA(serveCACert, serveCAKey)
	if err != nil {
		return fmt.Errorf("load CA: %w", err)
	}

	secret_map := get_secrets()

	services := []service.Service{
		service.NewRubyGems(secret_map["GEM_HOST_API_KEY"]),
		service.NewGitHub(secret_map["GH_TOKEN"]),
	}

	p := proxy.New(ca)
	p.InterceptRequest = func(req *http.Request) (*http.Response, bool) {
		resp, stop, svc := service.Process(services, req, authenticate)
		if svc != nil {
			if stop {
				log.Printf("Dropping request for %s\n", svc.Name())
				return resp, true
			}
			log.Printf("Allowing request for %s\n", svc.Name())
			return nil, false
		}
		log.Printf("No service matched request %s\n", req.URL.String())
		return nil, false
	}
	p.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNotFound {
			resp.StatusCode = 444
		}
		return nil
	}

	addr := fmt.Sprintf(":%d", servePort)
	log.Printf("listening on %s", addr)
	return http.ListenAndServe(addr, p)
}
