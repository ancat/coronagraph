package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"coronagraph/vault"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func get_passphrase() ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Passphrase: ")
	defer fmt.Fprintf(os.Stderr, "\n")
	return term.ReadPassword(int(os.Stdin.Fd()))
}

var localKeysCmd = &cobra.Command{
	Use:   "local-keys",
	Short: "Manage local key storage",
}

var localKeysInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize local key storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		var test vault.LocalVault

		passphrase, err := get_passphrase()
		defer clear(passphrase)
		if err != nil {
			panic(err)
		}

		err = test.Init(passphrase)
		if err != nil {
			panic(err)
		}

		err = test.WriteToFile("config-lv.yml")
		if err != nil {
			panic(err)
		}

		fmt.Printf("Initialized a local key store at config-lv.yml!\n")

		return nil
	},
}

var localKeysEditCmd = &cobra.Command{
	Use:	"edit",
	Short: 	"Edit local key store",
	RunE: func(cmd *cobra.Command, args []string) error {
		var test vault.LocalVault

		if err := test.LoadFromFile("config-lv.yml"); err != nil {
			panic(err)
		}


		tmpFile, err := os.CreateTemp("", "example-*.txt")
		if err != nil {
			fmt.Printf("Failed to create temp file: %v", err)
			return nil
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		editor_process := exec.Command("vim", tmpFile.Name())
		editor_process.Stdin = os.Stdin
		editor_process.Stdout = os.Stdout
		err = editor_process.Run()
		if err != nil {
			panic(err)
		}

		contents, err := os.ReadFile(tmpFile.Name())
		if strings.TrimSpace(string(contents)) == "" {
			fmt.Printf("Empty file; not writing any changes. Delete the file to clear its contents.\n")
			return nil
		}

		_, err = godotenv.Unmarshal(string(contents))
		if err != nil {
			fmt.Printf("Failed to parse secrets; it should be in dotenv format: %s\n", err)
			return nil
		}

		passphrase, err := get_passphrase()
		defer clear(passphrase)
		if err != nil {
			panic(err)
		}

		test.WriteData(passphrase, contents)

		err = test.WriteToFile("config-lv.yml")
		if err != nil {
			panic(err)
		}

		return nil
	},
}

var localKeysReadCmd = &cobra.Command{
	Use:	"read",
	Short: 	"View Key Store",
	RunE: func(cmd *cobra.Command, args []string) error {
		var test vault.LocalVault

		if err := test.LoadFromFile("config-lv.yml"); err != nil {
			panic(err)
		}


		passphrase, err := get_passphrase()
		defer clear(passphrase)
		if err != nil {
			panic(err)
		}

		data, err := test.ReadData(passphrase)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s\n", data)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(localKeysCmd)
	localKeysCmd.AddCommand(localKeysInitCmd)
	localKeysCmd.AddCommand(localKeysEditCmd)
	localKeysCmd.AddCommand(localKeysReadCmd)
}
