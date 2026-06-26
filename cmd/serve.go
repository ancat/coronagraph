package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"coronagraph/config"
	"coronagraph/proxy"
	"coronagraph/service"
	"coronagraph/vault"
)

var (
	servePort   int
	serveCACert string
	serveCAKey  string
	credentialSource  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the MITM proxy",
	RunE:  runServe,
}

func init() {
	cg_config, err := config.Load("config.yml")
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(serveCmd)

	servePort = cg_config.Port()
	serveCACert = cg_config.TLSCertificatePath()
	serveCAKey = cg_config.TLSKeyPath()
	credentialSource = cg_config.Credentials()
}

func get_local_secrets() map[string]string {
	var local_vault vault.LocalVault

	if err := local_vault.LoadFromFile("config-lv.yml"); err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "Passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	defer fmt.Fprintf(os.Stderr, "\n")
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

func get_op_secrets() map[string]string {
	contents, err := vault.ReadOP("op://Developer Creds/g4pixtjpdnd7btevhwf74pgw2u/notesPlain")
	if err != nil {
		panic(err)
	}

	secret_env, err := godotenv.Unmarshal(string(contents))
	if err != nil {
		panic(err)
	}

	return secret_env
}

func runServe(cmd *cobra.Command, args []string) error {
	ca, err := proxy.LoadCA(serveCACert, serveCAKey)
	if err != nil {
		return fmt.Errorf("load CA: %w", err)
	}

	var secret_map map[string]string

	switch credentialSource {
		case "local-vault":
			secret_map = get_local_secrets()
		case "1password":
			secret_map = get_op_secrets()
		default:
			panic("Unknown credential source")
	}

	services := []service.Service{
		service.NewRubyGems(secret_map["GEM_HOST_API_KEY"]),
		service.NewGitHub(secret_map["GH_TOKEN"]),
	}

	p := proxy.New(ca)
	p.InterceptRequest = func(req *http.Request) (*http.Response, bool) {
		resp, stop, svc := service.Process(services, req, authenticate)
		if svc != nil {
			if stop {
				log.Printf("[%s][DROPPED] %s %s\n", svc.Name(), req.Method, req.URL.Path)
				return resp, true
			}

			log.Printf("[%s][ALLOWED] %s %s\n", svc.Name(), req.Method, req.URL.Path)
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
