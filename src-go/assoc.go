package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    
    "golang.org/x/sys/windows/registry"
)

var initIconCache = sync.OnceFunc(func (){
    err := os.Mkdir(os.ExpandEnv(cfg().Assoc.IconCache), 0o666)
    if err != nil && !errors.Is(err, os.ErrExist) {
        panic(err)
    }
});

type Assoc struct {
    Name string
    Icon string
}

func getProgids(ext string) ([]string, error) {
    key, err := registry.OpenKey(registry.CLASSES_ROOT, ext, registry.READ)
    if err != nil {
        return nil, err
    }
    defer key.Close()
    ids := []string {}
    progid, _, err := key.GetStringValue("")
    if err == nil {
        ids = append(ids, progid)
    }
    subkey, err := registry.OpenKey(key, "OpenWithProgids", registry.READ)
    if err != nil {
        if len(ids) > 0{
            return ids, nil
        } else {
            return nil, err
        }
    }
    defer subkey.Close()
    names, err := subkey.ReadValueNames(0)
    if err != nil {
        if len(ids) > 0{
            return ids, nil
        } else {
            return nil, err
        }
    }
    ids = append(ids, names...)
    return ids, nil
}

func assocFromProgid(progid string) (*Assoc, error) {
    key, err := registry.OpenKey(registry.CLASSES_ROOT, progid, registry.READ)
    if err != nil {
        return nil, err
    }
    name, _, err := key.GetStringValue("")
    if err != nil {
        return nil, err
    }
    subkey, err := registry.OpenKey(key, "DefaultIcon", registry.READ)
    if err != nil {
        return nil, err
    }
    icon, _, err := subkey.GetStringValue("")
    if err != nil {
        return nil, err
    }
    return &Assoc { Name: name, Icon: icon }, nil
}

func GetFolderIcon() (string, error) {
    ok, path := tryGetIconCache("folder")
    if !ok {
        err := extractIcon("C:\\Windows\\system32\\imageres.dll", 3, path)
        if err != nil {
            return "", err
        }
    }
    return path, nil
}

func GetAssoc(name string) (*Assoc, error) {
    assoc, err := TryGetAssoc(name)
    if err != nil {
        assoc = &Assoc {}
        if len(filepath.Ext(name)) > 0 {
            assoc.Name = fmt.Sprintf("%s 文件", strings.ToUpper(filepath.Ext(name)[1:]))
        } else {
            assoc.Name = "文件"
        }
        ok, path := tryGetIconCache("file")
        if !ok {
            err := extractIcon("C:\\Windows\\system32\\imageres.dll", 2, path)
            if err != nil {
                return nil, err
            }
        }
        assoc.Icon = path
    }
    return assoc, nil
}

func TryGetAssoc(path string) (*Assoc, error) {
    cfg := cfg()
    for pat, assoc := range cfg.Assoc.Custom {
        if ok, err := filepath.Match(pat, path); err != nil {
            return nil, err
        } else if ok {
            return &assoc, nil
        }
    }
    ext := filepath.Ext(path)
    if len(ext) == 0 { 
        return nil, fmt.Errorf("cannot find association for files without extension") 
    }
    ids, err := getProgids(ext)
    if err != nil {
        return nil, err
    }
    for _, progid := range ids {
        assoc, err := assocFromProgid(progid)
        if err != nil {
            continue
        }
        cachename := ext
        if strings.Contains(assoc.Icon, "%1") {
            assoc.Icon = strings.ReplaceAll(assoc.Icon, "%1", path)
            cachename = Hash(path)
        }
        ok, name := tryGetIconCache(cachename)
        if ok {
            assoc.Icon = name
            return assoc, nil
        }
        if exe, index_s, ok := strings.Cut(assoc.Icon, ","); ok {
            exe, _ = strings.CutPrefix(exe, "\"")
            exe, _ = strings.CutSuffix(exe, "\"")
            var index int
            fmt.Sscanf(index_s, "%d", &index)
            if err := extractIcon(exe, index, name); err != nil {
                continue
            }
            assoc.Icon = name
        } else {
            exe, _ = strings.CutPrefix(exe, "\"")
            exe, _ = strings.CutSuffix(exe, "\"")
            if !strings.HasSuffix(exe, ".ico") {
                if err := extractIcon(path, 0, name); err != nil {
                    continue
                }
                assoc.Icon = name
            }
        }
        return assoc, nil
    }
    return nil, fmt.Errorf("no appropriate associations found")
}

func tryGetIconCache(symbol string) (bool, string) {
    initIconCache();
    name := filepath.Join(os.ExpandEnv(cfg().Assoc.IconCache), symbol + ".ico")
    _, err := os.Stat(name)
    if err != nil {
        return false, name
    }
    return true, name
}