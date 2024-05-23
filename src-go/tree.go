package main

import (
    "os"
    "path/filepath"
    "time"

    "gopkg.in/yaml.v3"
)

var Flags = U16Enum{ "undefined", "invisible", "visible", "readonly", "readwrite" };

type FileItem struct {
    Name    string         `json:"name"`
    Size     int64        `json:"size"`
    Time     time.Time    `json:"time"`
    Assoc    *string         `json:"assoc"`
    Flag    uint16        `json:"flag"`
    Effect    *Effect        `json:"effect"`
}

type DirItem struct {
    Name    string        `json:"name"`
    Time    time.Time    `json:"time"`
    Flag    uint16        `json:"flag"`
    Effect    *Effect        `json:"effect"`
}

type RuleItem struct {
    Flag    uint16
    Modes    []string
}

type Rules []RuleItem

func (r *Rules) FlagOf(name string) uint16 {
    for _, item := range *r {
        for _, mode := range item.Modes {
            if matched, _ := filepath.Match(mode, name); matched {
                return item.Flag
            }
        }
    }
    return Flags.Find("undefined")
}

type Effect struct {
    Definition    string    `json:"definition"`
    Direct        bool    `json:"direct"`
    Cause        string    `json:"cause"`
}

type Tree struct {
    prev    *Tree
    cache    map[string]*Tree
    Path    string // Absolute path for root, folder name for subtree.
    Rules    Rules
}

func CreateTree(path string) *Tree {
    if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
        return nil
    }
    tree := &Tree{
        prev: nil,
        cache: map[string]*Tree {},
        Path: path,
        Rules: nil,
    }
    tree.loadRules()
    return tree
}

func (t *Tree) IsRoot() bool {
    return t.prev == nil
}

func (t *Tree) Root() *Tree {
    var p *Tree
    for p = t; !p.IsRoot(); p = p.prev {}
    return p
}

func (t *Tree) Next(name string) *Tree {
    for namecache, treecache := range t.cache {
        if namecache == name {
            return treecache
        }
    }
    if stat, err := os.Stat(filepath.Join(t.AbsPath(), name)); err != nil || !stat.IsDir() {
        return nil
    }
    if flag, _ := t.FlagOf(name); flag <= Flags.Find("invisible") && !cfg().Tree.ShowHidden {
        return nil
    }
    tree := &Tree{
        prev: t,
        cache: map[string]*Tree {},
        Path: name,
        Rules: nil,
    }
    tree.loadRules()
    if cfg().Tree.CachePolicy != "never" {
        t.cache[name] = tree
    }
    return tree
}

func (t *Tree) Reload() {
    t.loadRules()
    for _, tree := range t.cache {
        tree.loadRules()
    }
}

func (t *Tree) RelPath(to *Tree) string {
    s := ""
    for p := t; p != to; p = p.prev {
        s = filepath.Join(p.Path, s)
    }
    return s
}

func (t *Tree) AbsPath() string {
    return filepath.Join(t.Root().Path, t.RelPath(t.Root()))
}

func (t *Tree) loadRules() {
    rulesfile := filepath.Join(t.AbsPath(), cfg().Tree.RulesFile)
    rulesbin, err := os.ReadFile(rulesfile)
    if err != nil { return }
    rules := map[string]([]string) {}
    yaml.Unmarshal(rulesbin, &rules)
    for flag, modes := range rules {
        if ok, flagvalue := Flags.TryFind(flag); ok {
            t.Rules = append(t.Rules, RuleItem{ Flag: flagvalue, Modes: modes })
        }
    }
}

func (t *Tree) FlagOf(name string) (uint16, *Effect) {
    for p := t; p != nil; p = p.prev {
        if mode := p.Rules.FlagOf(filepath.Join(t.RelPath(p), name)); mode != Flags.Find("undefined") {
            return mode, &Effect{
                Definition: filepath.Join(p.RelPath(p.Root()), cfg().Tree.RulesFile),
                Direct: true,
                Cause: "",
            }
        }
    }
    for p := t; !p.IsRoot(); p = p.prev {
        for q := p.prev; q != nil; q = q.prev {
            if mode := q.Rules.FlagOf(filepath.Join(p.prev.RelPath(q), p.Path)); mode != Flags.Find("undefined") {
                return mode, &Effect{
                    Definition: filepath.Join(q.RelPath(q.Root()), cfg().Tree.RulesFile),
                    Direct: false,
                    Cause: p.RelPath(p.Root()),
                }
            }
        }
    }
    return Flags.Find(cfg().Tree.DefaultFlag), nil
}

func (t *Tree) Scan() ([]FileItem, []DirItem, error) {
    entries, err := os.ReadDir(t.AbsPath())
    if err != nil {
        return nil, nil, err
    }
    files := []FileItem{}
    dirs := []DirItem{}
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            return nil, nil, err
        }
        flag, effect := t.FlagOf(entry.Name())
        if !cfg().Tree.ShowHidden && flag <= Flags.Find("invisible") { continue }
        if entry.IsDir() {
            dirs = append(dirs, DirItem{
                Name: entry.Name(),
                Time: info.ModTime(),
                Flag: flag,
                Effect: effect,
            })
        } else {
            assoc, _ := GetAssoc(entry.Name())
            files = append(files, FileItem{
                Name: entry.Name(),
                Size: info.Size(),
                Time: info.ModTime(),
                Assoc: &assoc.Name,
                Flag: flag,
                Effect: effect,
            })
        }
    }
    return files, dirs, nil
}