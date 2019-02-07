package main

import (
	"fmt"
	"os"

	"github.com/nacamp/go-simplechain/container"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/console"
	"github.com/nacamp/go-simplechain/crypto"

	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/log"
	"github.com/urfave/cli"
)

func run(c *cli.Context) {
	log.Init("", log.InfoLevel, 0)
	if c.String("config") == "" {
		log.CLog().Fatal("not found config")
		return
	}
	config := cmd.NewConfigFromFile(c.String("config"))
	ns := container.NewNodeServer(config)
	ns.Start()
	select {}

}

func AccountImportAction(c *cli.Context) {
	if c.String("config") == "" {
		log.CLog().Fatal("not found config")
		return
	}
	config := cmd.NewConfigFromFile(c.String("config"))
	// if len(c.Args()) < 1 {
	// 	log.CLog().Fatal("need privatekey passphrase")
	// }
	privateKey := getPrivateKey()
	passphrase := getPassPhrase("", true, 0, []string{})
	accountImportAction(config.KeystoreFile, privateKey, passphrase)
}

func accountImportAction(path string, priv string, passphrase string) {
	//TODO: priv validate
	key := new(account.Key)
	key.PrivateKey = crypto.ByteToPrivateKey(common.FromHex(priv))
	key.Address = crypto.CreateAddressFromPrivateKey(key.PrivateKey)
	fmt.Printf("address : %v\n", common.AddressToHex(key.Address))
	wallet := account.NewWallet(path)
	wallet.Load()
	wallet.StoreKey(key, passphrase)
}

func AccountNewAction(c *cli.Context) {
	if c.String("config") == "" {
		log.CLog().Fatal("not found config")
		return
	}
	config := cmd.NewConfigFromFile(c.String("config"))
	passphrase := getPassPhrase("", true, 0, []string{})
	accountNewAction(config.KeystoreFile, passphrase)
}

func accountNewAction(path string, passphrase string) {
	//TODO: priv validate
	priv, address := crypto.CreateAddress()
	key := new(account.Key)
	key.PrivateKey = priv
	key.Address = address
	fmt.Printf("address : %v\n", common.AddressToHex(key.Address))
	wallet := account.NewWallet(path)
	wallet.Load()
	wallet.StoreKey(key, passphrase)
}

func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func getPrivateKey() string {
	privateKey, err := console.Stdin.PromptPassword("PrivateKey: ")
	if err != nil {
		utils.Fatalf("Failed to read privateKey: %v", err)
	}
	return privateKey
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "",
			Usage: "config file path",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "account",
			Usage: "account import|new ...",
			Subcommands: []cli.Command{
				{
					Name:        "import",
					Flags:       app.Flags,
					Usage:       "import privatekey",
					ArgsUsage:   "<privatekey>",
					Action:      AccountImportAction,
					Category:    "ACCOUNT COMMANDS",
					Description: `demo account import`,
				},
				{
					Name:        "new",
					Flags:       app.Flags,
					Usage:       "new",
					Action:      AccountNewAction,
					Category:    "ACCOUNT COMMANDS",
					Description: `demo account new`,
				},
			},
		},
	}
	app.Action = run
	err := app.Run(os.Args)
	if err != nil {
		log.CLog().Fatal(err)
	}
}
