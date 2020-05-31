package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bonedaddy/krabtech-proxy/internal/proxy"
	"github.com/urfave/cli/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "listen.address",
			Aliases: []string{"la"},
			Usage:   "address for proxy to listsen on",
			Value:   "localhost:8448",
		},
		&cli.BoolFlag{
			Name:  "https.enabled",
			Usage: "enable listening with https",
		},
	}
	app.Commands = cli.Commands{
		&cli.Command{
			Name: "run",
			Action: func(c *cli.Context) error {
				p := proxy.New(
					c.String("listen.address"),
					map[string]*proxy.BackendHost{
						"localhost": &proxy.BackendHost{
							Addr:     "localhost:8080",
							Insecure: true,
						},
					},
				)
				return p.Run(ctx)
			},
		},
		&cli.Command{
			Name: "test",
			Action: func(c *cli.Context) error {
				req, err := http.NewRequest("POST", "http://"+c.String("listen.address"), nil)
				if err != nil {
					return err
				}
				req.Host = "localhost"
				client := http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}