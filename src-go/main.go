package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"golang.org/x/sys/windows/registry"
)

//go:generate rsrc -ico ../public/favicon.ico
//go:generate go-bindata-assetfs -o assets.go -nomemcopy ../dist/...

func main() {
    defer func() {
        if err := recover(); err != nil {
            fmt.Println("error:", err)
        }
    }()
    if len(os.Args) < 2 {
        fmt.Printf("Usage: %s [init|serve]\n", os.Args[0])
        os.Exit(2)
    }
    switch os.Args[1] {
    case "serve": {
        fs := flag.NewFlagSet("", flag.ExitOnError)
        fs.StringVar(&cfgPath, "config", os.ExpandEnv(".\\sagasu-config.toml;${USERPROFILE}\\sagasu-config.toml"), "A semicolon-separated list of config file locations.")
        phost := fs.String("host", "", "Host to bind to.")
        pport := fs.Int("port", 0, "Port to bind to.")
        proot := fs.String("root", ".", "Root directory to serve.")
        fs.Parse(os.Args[2:])
        var host string 
        var port int
        if len(*phost) > 0 {
            host = *phost
        } else {
            host = cfg().Http.Host
        }
        if *pport > 0 {
            port = *pport
        } else {
            port = cfg().Http.Port
        }
        NewServer(*proot)(host, port)
        break
    }
    case "init": {
        basekey, err := registry.OpenKey(registry.CLASSES_ROOT, "Directory", registry.ALL_ACCESS)
        if err != nil {
            panic(fmt.Errorf("failed to open base registry key: %v", err))
        }
        
        shellkey, exists, err := registry.CreateKey(basekey, "shell\\Sagasu", registry.ALL_ACCESS)
        if exists {
            break
        }
        if err != nil {
            panic(fmt.Errorf("failed to open shell registry key: %v", err))
        }
        defer shellkey.Close()

        shellkey.SetStringValue("", "使用 Sagasu 共享")
        extractIcon(os.Args[0], 0, "sagasu-icon.ico")
        shellkey.SetStringValue("Icon", filepath.Join(filepath.Dir(os.Args[0]), "sagasu-icon.ico"))

        cmdkey, _, err := registry.CreateKey(shellkey, "command", registry.ALL_ACCESS)
        if err != nil {
            panic(fmt.Errorf("failed to open shell registry key: %v", err))
        }
        defer cmdkey.Close()

        cmdkey.SetStringValue("", fmt.Sprintf("\"%s\" serve --root \"%%V\"", os.Args[0]))
        fmt.Println("Registry updated.")

        fp, err := os.Create(os.ExpandEnv("${USERPROFILE}\\sagasu-config.toml"))
        if err != nil {
            panic(fmt.Errorf("failed to create configuration file: %v", err))
        }
        toml.NewEncoder(fp).Encode(defConfig)
        fmt.Println("Successfully created configuration file at $USERPROFILE.")
        break
    }
    }
}