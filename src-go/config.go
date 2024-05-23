package main

import (
    "fmt"
    "strings"

    "github.com/BurntSushi/toml"
)

type AssocSection struct {
    Custom    map[string]Assoc
    IconCache string
}

type TreeSection struct {
    DefaultFlag    string
    RulesFile    string
    ShowHidden    bool
    CachePolicy    string
}

type HttpSection struct {
    Host    string
    Port    int
    Debug    bool
}

type Config struct {
    Assoc    AssocSection
    Tree    TreeSection
    Http    HttpSection
}

var cfgCache *Config

var cfgPath string

var defConfig = Config{
    Assoc: AssocSection{
        Custom: map[string]Assoc{},
        IconCache: "${USERPROFILE}\\.sagasu-icon-cache",
    },
    Tree: TreeSection{
        DefaultFlag: "readonly",
        RulesFile: ".rules.yml",
        ShowHidden: false,
        CachePolicy: "always",
    },
    Http: HttpSection{
        Host: "0.0.0.0",
        Port: 8080,
        Debug: false,
    },
}

func cfg() *Config {
    if cfgCache == nil {
        cfgCache = new(Config)
        loaded := false
        for _, path := range strings.Split(cfgPath, ";") {
            _, err := toml.DecodeFile(path, cfgCache)
            if err == nil { 
                loaded = true
                break 
            }
        }
        if !loaded {
            fmt.Println("Warning: cannot load config, using default setup.")
            cfgCache = &defConfig
        }
    }
    return cfgCache
}