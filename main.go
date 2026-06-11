package main

import (
	_ "embed"
	"fmt"
	"image"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

//go:embed fonts/DejaVuSans.ttf
var fontData []byte

const (
	inputPNG  = "mkk.png"
	boxX, boxY = 70, 320
	boxW, boxH = 370.0, 140.0
)

var (
	fontPath  string
	baseImage image.Image
)

func init() {
	f, err := os.CreateTemp("", "overlay_font_*.ttf")
	if err != nil {
		log.Fatal(err)
	}
	fontPath = f.Name()
	if _, err := f.Write(fontData); err != nil {
		log.Fatal(err)
	}
	f.Close()

	baseImage, err = gg.LoadImage(inputPNG)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := gin.Default()
	r.GET("/overlay", handleOverlay)
	log.Println("servidor em :8089")
	log.Fatal(r.Run(":8089"))
}

var ogCrawlers = []string{
	"facebookexternalhit", "Facebot",
	"WhatsApp", "TelegramBot",
	"Twitterbot", "Slackbot",
	"Discordbot", "LinkedInBot",
	"SkypeUriPreview",
}

func wantsHTML(c *gin.Context) bool {
	for _, part := range strings.Split(c.GetHeader("Accept"), ",") {
		if strings.TrimSpace(part) == "text/html" {
			return true
		}
	}
	ua := c.GetHeader("User-Agent")
	for _, bot := range ogCrawlers {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

func handleOverlay(c *gin.Context) {
	text := c.Query("text")
	if text == "" {
		c.String(http.StatusBadRequest, "missing text param")
		return
	}

	if c.Query("raw") == "" && wantsHTML(c) {
		servePreview(c, text)
		return
	}

	dc := gg.NewContextForImage(baseImage)

	fontSize := findOptimalSize(dc, fontPath, text, boxW, boxH)
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		c.String(http.StatusInternalServerError, "font error")
		return
	}

	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored(text, boxX+boxW/2, boxY+boxH/2, 0.5, 0.5)

	c.Header("Content-Type", "image/png")
	dc.EncodePNG(c.Writer)
}

func servePreview(c *gin.Context, text string) {
	scheme := "https"
	host := c.Request.Host
	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
		scheme = "http"
	}
	imageURL := fmt.Sprintf("%s://%s%s&raw=1", scheme, host, c.Request.URL.String())

	html := fmt.Sprintf(`<!DOCTYPE html>
<html prefix="og: https://ogp.me/ns#">
<head>
<meta charset="utf-8">
<title>mkk</title>
<meta property="og:title" content="mkk">
<meta property="og:image" content="%s">
<meta property="og:image:type" content="image/png">
<meta property="og:image:width" content="512">
<meta property="og:image:height" content="512">
<meta property="og:type" content="website">
<meta name="twitter:card" content="summary_large_image">
</head>
<body style="margin:0;background:#000;display:flex;justify-content:center;align-items:center;min-height:100vh">
<img src="%s" alt="overlay" style="max-width:100%%;height:auto">
</body>
</html>`, imageURL, imageURL)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func findOptimalSize(dc *gg.Context, fontPath, text string, maxW, maxH float64) float64 {
	size := maxH
	for range 10 {
		if err := dc.LoadFontFace(fontPath, size); err != nil {
			break
		}
		w, h := dc.MeasureString(text)
		if w <= maxW && h <= maxH {
			return size
		}
		size *= math.Min(maxW/w, maxH/h) * 0.95
	}
	return size
}
