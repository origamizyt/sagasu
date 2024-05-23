package main

import (
    "encoding/binary"
    "fmt"
    "hash/adler32"
    "net"
    "os"
    "syscall"
    "unsafe"

    "github.com/gonutz/w32"
)

type U16Enum []string

func CreateU16Enum(members ...string) U16Enum {
    return members
}

func (enum *U16Enum) Get(value uint16) string {
    return (*enum)[value]
}

func (enum *U16Enum) Find(name string) uint16 {
    for i, member := range *enum {
        if member == name { return uint16(i) }
    }
    panic(fmt.Errorf("invalid enum member: %s", name))
}

func (enum *U16Enum) TryFind(name string) (bool, uint16) {
    for i, member := range *enum {
        if member == name { return true, uint16(i) }
    }
    return false, 0
}

func Hash(name string) string {
    return fmt.Sprintf("%.8x", adler32.Checksum([]byte(name)));
}

func getIP() string {
    dial, err := net.Dial("udp", "8.8.8.8:domain")
    if err != nil {
        panic(fmt.Errorf("cannot get ip address: %v", err))
    }
    defer dial.Close()
    return dial.LocalAddr().(*net.UDPAddr).IP.String()
}


type _ICONDIR struct {
    Reserved uint16
    Type     uint16
    Count    uint16
}

type _ICONDIRENTRY struct {
    Width       byte
    Height      byte
    ColorCount  byte
    Reserved    byte
    Planes      uint16
    BitCount    uint16
    BytesInRes  uint32
    ImageOffset uint32
}

type _ICONINFO struct {
    FIcon       uint32
    XHotspot    uint32
    YHotspot    uint32
    HbmMask     uintptr
    HbmColor    uintptr
}

func extractIcon(exe string, index int, ico string) error {
    hIcon := w32.ExtractIcon(exe, index)
    defer w32.DestroyIcon(hIcon)
    return saveIconToFile(hIcon, ico)
}

func saveIconToFile(hIcon w32.HICON, filePath string) error {
    // Get icon information
    user32 := syscall.NewLazyDLL("user32.dll")
    GetIconInfo := user32.NewProc("GetIconInfo")

    var iconInfo _ICONINFO
    if res, _, _ := GetIconInfo.Call(uintptr(hIcon), uintptr(unsafe.Pointer(&iconInfo))); res == 0 {
        return fmt.Errorf("failed to get icon info")
    }
    defer w32.DeleteObject(w32.HGDIOBJ(iconInfo.HbmColor))
    defer w32.DeleteObject(w32.HGDIOBJ(iconInfo.HbmMask))

    // Get bitmap information
    colorBitmapInfo := w32.BITMAP{}
    if res := w32.GetObject(w32.HGDIOBJ(iconInfo.HbmColor), unsafe.Sizeof(colorBitmapInfo), unsafe.Pointer(&colorBitmapInfo)); res == 0 {
        return fmt.Errorf("failed to get color bitmap info")
    }

    width := int(colorBitmapInfo.BmWidth)
    height := int(colorBitmapInfo.BmHeight)
    colorDepth := int(colorBitmapInfo.BmBitsPixel)

    // Get the DIB bits for the color and mask bitmaps
    colorImageSize := ((width * colorDepth + 31) / 32) * 4 * height
    maskImageSize := ((width * 1 + 31) / 32) * 4 * height
    colorImage := make([]byte, colorImageSize)
    maskImage := make([]byte, maskImageSize)

    hdc := w32.GetDC(0)
    defer w32.ReleaseDC(0, hdc)

    // Fill BITMAPINFO structures
    colorBmi := w32.BITMAPINFO{
        BmiHeader: w32.BITMAPINFOHEADER{
            BiSize:    uint32(unsafe.Sizeof(w32.BITMAPINFOHEADER{})),
            BiWidth:   int32(width),
            BiHeight:  int32(height),
            BiPlanes:  1,
            BiBitCount: uint16(colorDepth),
            BiCompression: w32.BI_RGB,
        },
    }

    maskBmi := w32.BITMAPINFO{
        BmiHeader: w32.BITMAPINFOHEADER{
            BiSize:    uint32(unsafe.Sizeof(w32.BITMAPINFOHEADER{})),
            BiWidth:   int32(width),
            BiHeight:  int32(height),
            BiPlanes:  1,
            BiBitCount: 1,
            BiCompression: w32.BI_RGB,
        },
    }

    // Get DIB bits
    if res := w32.GetDIBits(hdc, w32.HBITMAP(iconInfo.HbmColor), 0, uint(height), unsafe.Pointer(&colorImage[0]), &colorBmi, w32.DIB_RGB_COLORS); res == 0 {
        return fmt.Errorf("failed to get color bitmap bits")
    }
    if res := w32.GetDIBits(hdc, w32.HBITMAP(iconInfo.HbmMask), 0, uint(height), unsafe.Pointer(&maskImage[0]), &maskBmi, w32.DIB_RGB_COLORS); res == 0 {
        return fmt.Errorf("failed to get mask bitmap bits")
    }

    // Create and write the ICO file
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
    }
    defer file.Close()

    // Write ICONDIR
    iconDir := _ICONDIR{
        Reserved: 0,
        Type:     1,
        Count:    1,
    }
    if err := binary.Write(file, binary.LittleEndian, &iconDir); err != nil {
        return fmt.Errorf("failed to write icon dir: %v", err)
    }

    // Write ICONDIRENTRY
    iconDirEntry := _ICONDIRENTRY{
        Width:       byte(width),
        Height:      byte(height),
        ColorCount:  0,
        Reserved:    0,
        Planes:      1,
        BitCount:    uint16(colorDepth),
        BytesInRes:  uint32(40 + len(colorImage) + len(maskImage)),
        ImageOffset: uint32(6 + 16),
    }
    if err := binary.Write(file, binary.LittleEndian, &iconDirEntry); err != nil {
        return fmt.Errorf("failed to write icon dir entry: %v", err)
    }

    // Write BITMAPINFOHEADER
    colorBmi.BmiHeader.BiHeight = int32(height * 2) // height of color + mask
    if err := binary.Write(file, binary.LittleEndian, &colorBmi.BmiHeader); err != nil {
        return fmt.Errorf("failed to write bitmap info header: %v", err)
    }

    // Write color image data
    if _, err := file.Write(colorImage); err != nil {
        return fmt.Errorf("failed to write color image: %v", err)
    }

    // Write mask image data
    if _, err := file.Write(maskImage); err != nil {
        return fmt.Errorf("failed to write mask image: %v", err)
    }

    return nil
}

