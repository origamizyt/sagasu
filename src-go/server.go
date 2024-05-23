package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mdp/qrterminal/v3"
	"golang.org/x/crypto/blake2b"
)

type Server func(host string, port int)

func NewServer(root string) Server {
    if (!cfg().Http.Debug) {
        gin.SetMode(gin.ReleaseMode)
    }

    app := gin.New()
    app.Use(gin.Recovery())

    tree := CreateTree(root)
    upgrader := websocket.Upgrader{
        ReadBufferSize: 1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    if tree == nil {
        panic(fmt.Errorf("cannot open directory: %s", root))
    }

    app.Use(cors.New(cors.Config{
        AllowAllOrigins: true,
        AllowMethods: []string { "GET", "OPTIONS" },
        AllowHeaders: []string { "Origin", "Content-Length", "Content-Type" },
    }))
    
    fs := assetFS()
    for _, name := range AssetNames() {
        name, _ = strings.CutPrefix(name, "../dist/")
        app.StaticFileFS("/" + name, "dist/" + name, fs)
    }

    app.GET("/", func (c *gin.Context) {
        c.FileFromFS("dist/", fs)
    })

    app.GET("/tree/*path", func (c *gin.Context) {
        path := c.Param("path")
        path, _ = strings.CutPrefix(path, "/")
        t := tree
        if len(path) > 0 {
            for _, segment := range strings.Split(path, "/") {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        files, dirs, err := t.Scan()
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
                "error": err.Error(),
            })
            return
        }
        c.JSON(http.StatusOK, gin.H {
            "ok": true,
            "data": gin.H {
                "files": files,
                "dirs": dirs,
            },
        })
    })

    app.GET("/fileicon/*path", func (c *gin.Context) {
        path := c.Param("path")
        path, _ = strings.CutPrefix(path, "/")
        path = strings.ReplaceAll(path, "/", "\\")
        path, _ = filepath.Abs(filepath.Join(tree.Path, path))
        assoc, err := GetAssoc(path)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }
        c.Header("Cache-Control", "no-cache")
        c.File(assoc.Icon)
    })

    app.GET("/foldericon", func (c *gin.Context) {
        path, err := GetFolderIcon()
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        } 
        c.File(path)
    })

    app.GET("/file/*path", func (c *gin.Context) {
        path := c.Param("path")
        download := c.Query("download")
        path, _ = strings.CutPrefix(path, "/")
        parts := strings.Split(path, "/")
        t := tree
        if len(path) > 0 {
            for _, segment := range parts[:len(parts)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        loc := filepath.Join(t.AbsPath(), parts[len(parts)-1])
        _, err := os.Stat(loc)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusNotFound, gin.H {
                "ok": false,
                "error": parts[len(parts)-1],
            })
            return
        }
        flag, _ := t.FlagOf(parts[len(parts)-1])
        if flag <= Flags.Find("invisible") && !cfg().Tree.ShowHidden {
            c.AbortWithStatusJSON(http.StatusNotFound, gin.H {
                "ok": false,
                "error": parts[len(parts)-1],
            })
            return
        } else if flag <= Flags.Find("visible") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": parts[len(parts)-1],
            })
            return
        }
        if download == "true" {
            c.Header("Content-Disposition", "attachment; filename=\"" + parts[len(parts)-1] + "\"")
        }
        c.File(loc)
    })

    app.GET("/upload/*path", func (c *gin.Context) {
        path := c.Param("path")
        path, _ = strings.CutPrefix(path, "/")
        parts := strings.Split(path, "/")
        t := tree
        if len(path) > 0 {
            for _, segment := range parts[:len(parts)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ := t.FlagOf(parts[len(parts)-1])
        if flag < Flags.Find("readwrite") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": parts[len(parts)-1],
            })
            return
        }
        loc := filepath.Join(t.AbsPath(), parts[len(parts)-1])
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            c.AbortWithStatus(http.StatusBadRequest)
            return
        }

        body := struct {
            Count    int        `json:"count"`
            Key        string    `json:"key"`
        }{}
        err = conn.ReadJSON(&body)
        defer conn.Close()

        if err != nil {
            conn.WriteControl(
                websocket.CloseMessage, 
                websocket.FormatCloseMessage(websocket.ClosePolicyViolation, ""), 
                time.Time{},
            )
            return
        }
        key, _ := hex.DecodeString(body.Key)
        tmp, _ := os.CreateTemp("", "")
        conn.WriteJSON(true)

        for i := 0; i < body.Count; i++ {
            for {
                _, data, err := conn.ReadMessage()
                if err != nil {
                    conn.WriteControl(
                        websocket.CloseMessage, 
                        websocket.FormatCloseMessage(websocket.ClosePolicyViolation, ""), 
                        time.Time{},
                    )
                    tmp.Close()
                    os.Remove(tmp.Name())
                    return
                }
                sig, data := data[:32], data[32:]
                hasher, _ := blake2b.New(32, key)
                hasher.Write(data)
                if slices.Equal(sig, hasher.Sum(nil)) {
                    tmp.Write(data)
                    conn.WriteJSON(true)
                    break
                }
                conn.WriteJSON(false)
            }
        }

        tmp.Close()
        err = os.Remove(loc)
        if err != nil {
            conn.WriteControl(
                websocket.CloseMessage, 
                websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""), 
                time.Time{},
            )
            return
        }
        err = os.Rename(tmp.Name(), loc)
        if err != nil {
            conn.WriteControl(
                websocket.CloseMessage, 
                websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""), 
                time.Time{},
            )
            return
        }

        conn.WriteControl(
            websocket.CloseMessage, 
            websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), 
            time.Time{},
        )

        if cfg().Tree.CachePolicy == "upload" {
            tree.Reload()
        }
    })

    app.POST("/move", func (c *gin.Context) {
        body := struct {
            From    []string    `json:"from"`
            To      []string    `json:"to"`
        }{}
        err := c.BindJSON(&body)
        if err != nil { return }
        
        t := tree
        if len(body.From) > 1 {
            for _, segment := range body.From[:len(body.From)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ := t.FlagOf(body.From[len(body.From)-1])
        if flag < Flags.Find("readwrite") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": body.From[len(body.From)-1],
            })
            return
        }
        from_loc := filepath.Join(t.AbsPath(), body.From[len(body.From)-1])

        t = tree
        if len(body.To) > 1 {
            for _, segment := range body.To[:len(body.To)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ = t.FlagOf(body.To[len(body.To)-1])
        if flag < Flags.Find("readwrite") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": body.To[len(body.To)-1],
            })
            return
        }
        to_loc := filepath.Join(t.AbsPath(), body.To[len(body.To)-1])

        err = os.Rename(from_loc, to_loc)

        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }

        c.JSON(http.StatusOK, gin.H {
            "ok": true,
        })
    })

    app.POST("/copy", func (c *gin.Context) {
        body := struct {
            From    []string    `json:"from"`
            To      []string    `json:"to"`
        }{}
        err := c.BindJSON(&body)
        if err != nil { return }
        
        t := tree
        if len(body.From) > 1 {
            for _, segment := range body.From[:len(body.From)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ := t.FlagOf(body.From[len(body.From)-1])
        if flag < Flags.Find("readonly") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": body.From[len(body.From)-1],
            })
            return
        }
        from_loc := filepath.Join(t.AbsPath(), body.From[len(body.From)-1])

        t = tree
        if len(body.To) > 1 {
            for _, segment := range body.To[:len(body.To)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ = t.FlagOf(body.To[len(body.To)-1])
        if flag < Flags.Find("readwrite") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": body.To[len(body.To)-1],
            })
            return
        }
        to_loc := filepath.Join(t.AbsPath(), body.To[len(body.To)-1])

        src, err := os.Open(from_loc)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }
        defer src.Close()
        dst, err := os.Create(to_loc)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }
        defer dst.Close()

        _, err = io.Copy(dst, src)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }

        c.JSON(http.StatusOK, gin.H {
            "ok": true,
        })
    })

    app.POST("/delete/*path", func (c *gin.Context) {
        path := c.Param("path")
        path, _ = strings.CutPrefix(path, "/")
        parts := strings.Split(path, "/")
        t := tree
        if len(path) > 0 {
            for _, segment := range parts[:len(parts)-1] {
                t = t.Next(segment)
                if t == nil {
                    c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                        "ok": false,
                        "error": segment,
                    })
                    return
                }
            }
        }
        flag, _ := t.FlagOf(parts[len(parts)-1])
        if flag < Flags.Find("readwrite") {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H {
                "ok": false,
                "error": parts[len(parts)-1],
            })
            return
        }

        loc := filepath.Join(t.AbsPath(), parts[len(parts)-1])
        err := os.Remove(loc)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H {
                "ok": false,
            })
            return
        }

        c.JSON(http.StatusOK, gin.H {
            "ok": true,
        })
    })

    return func(host string, port int) {
        fmt.Println("探す (Sagasu) - Lightweight Remote File System")
        fmt.Printf("Serving root: %s\n\n", root)

        if host == "0.0.0.0" {
            ip := getIP()
            qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s:%d", ip, port), qrterminal.M, os.Stdout)
            fmt.Println()
            fmt.Printf("Local Endpoint @ http://127.0.0.1:%d\n", port)
            fmt.Printf("Public Endpoint @ http://%s:%d", ip, port)
        } else {
            fmt.Printf("Endpoint @ http://%s:%d", host, port)
        }

        app.Run(fmt.Sprintf("%s:%d", host, port))
    }
}