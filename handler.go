package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DusanKasan/parsemail"
	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	DeviceWeb     = "web"
	DeviceiPhone  = "iphone"
	DeviceOutlook = "outlook"
	DeviceGmail   = "gmail"
)

var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
var chromePath = ""

func randRunes(n int, source []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = source[rand.Intn(len(source))]
	}
	return string(b)
}

func RandText(n int) string {
	return randRunes(n, letterRunes)
}

func prepareStoreDir(storeDir string) error {
	st, err := os.Stat(storeDir)
	if err != nil {
		log.Println("Prepare storedir:", storeDir)
		return os.MkdirAll(storeDir, fs.ModePerm)
	}

	if !st.IsDir() {
		return fmt.Errorf("storedir: %s is not directory", storeDir)
	}
	return nil
}

func cleanMail(mailid string) {
	filepath.Walk(storeDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, mailid) {
			os.Remove(path)
		}
		return nil
	})
}

func loadTemplate(device string) string {
	fname := "html/" + device + ".html"
	data, err := os.ReadFile(fname)
	if err != nil {
		log.Println("load tempalte fail", fname, err)
		return ""
	}
	return string(data)
}

type RenderRequest struct {
	Waitload  int     `json:"waitload" form:"waitload" query:"waitload"`
	Format    string  `json:"format" form:"format" query:"format"`
	Device    string  `json:"device" form:"device" query:"device"`
	Headless  string  `json:"headless" form:"headless" query:"headless"`
	Textonly  bool    `json:"textonly" form:"textonly" query:"textonly"`
	Timezone  string  `json:"tz" form:"tz" query:"tz"`
	Author    string  `json:"author" form:"author" query:"author"`
	Watermark string  `json:"watermark" form:"watermark" query:"watermark"`
	Content   string  `json:"content,omitempty"`
	ViewPort  string  `json:"viewport" form:"viewport" query:"viewport"` // '0,0,800,800'
	Scale     float64 `json:"scale" form:"scale" query:"scale"`
	WithHiDPI bool    `json:"hidpi" form:"hidpi" query:"hidpi"`
}

type Attachment struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path"`
	Size int    `json:"size"`
}

type Mail struct {
	Size    int
	Subject string
	Sender  *mail.Address
	From    *mail.Address
	ReplyTo []*mail.Address
	To      []*mail.Address
	Cc      []*mail.Address
	Bcc     []*mail.Address

	Date        string
	HTMLBody    string
	TextBody    string
	Attachments []Attachment
}

func RegisterHandlers(r *gin.Engine) {
	r.POST("/mailrender", handleMailrender)
	r.StaticFS("/_/", http.Dir(storeDir))

	r.StaticFile("/", "html/index.html")
	r.StaticFS("/static", http.Dir("html/static"))
}

func handleMailrender(c *gin.Context) {
	var req RenderRequest
	err := c.BindQuery(&req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err = c.Bind(&req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if strings.Contains(c.ContentType(), "multipart/form-data") {
		if content, err := c.FormFile("content"); err == nil {
			if f, err := content.Open(); err == nil {
				data, _ := io.ReadAll(f)
				req.Content = string(data)
			}
		}
	}

	if req.Content == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("`content` is empty"))
		return
	}

	if req.Device == "" {
		req.Device = DeviceWeb
	}

	if req.Author == "" {
		req.Author = author
	}

	if req.Format == "" {
		req.Format = "png" // Default is image/png
	}

	r := bytes.NewReader([]byte(req.Content))
	msg, err := parsemail.Parse(r)
	if err != nil {
		log.Println("parse mail fail", err)
		c.AbortWithError(http.StatusBadRequest, errors.New("parse mail fail "+err.Error()))
		return
	}

	viewPort := req.ViewPort
	var floatVps []float64
	if viewPort != "" {
		vp := strings.Split(viewPort, ",")
		if len(vp) != 4 {
			log.Println("invalid viewport", viewPort)
			c.AbortWithError(http.StatusInternalServerError, errors.New("invalid view port "+viewPort))
			return
		}

		for _, v := range vp {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				log.Println("invalid viewport", viewPort, err)
				c.AbortWithError(http.StatusInternalServerError, errors.New("invalid view port "+viewPort))
				return
			}
			floatVps = append(floatVps, f)
		}
	}

	if req.Scale <= 0 || req.Scale >= 2 {
		req.Scale = 1
	}

	mail := Mail{
		Size:     len(req.Content),
		Subject:  msg.Subject,
		Sender:   msg.Sender,
		ReplyTo:  msg.ReplyTo,
		To:       msg.To,
		Cc:       msg.Cc,
		Bcc:      msg.Bcc,
		HTMLBody: msg.HTMLBody,
		TextBody: msg.TextBody,
	}

	if req.Timezone != "" {
		tz, _ := time.LoadLocation(req.Timezone)
		if tz != nil {
			msg.Date = msg.Date.In(tz)
		}
	}

	mail.Date = msg.Date.Format("2006-01-02 03:04:05")
	if len(msg.From) > 0 {
		mail.From = msg.From[0]
	}
	//extract mail
	for _, att := range msg.Attachments {
		attData, err := io.ReadAll(att.Data)
		if err == nil {
			continue
		}
		mail.Attachments = append(mail.Attachments, Attachment{
			Name: att.Filename,
			Size: len(attData),
		})
	}

	mailid := RandText(20)
	defer func() {
		cleanMail(mailid)
	}()

	htmlbody := msg.HTMLBody
	if htmlbody == "" {
		htmlbody = msg.TextBody
	}

	for _, efile := range msg.EmbeddedFiles {
		storepath := fmt.Sprintf("%s-%s", mailid, efile.CID)
		attData, err := io.ReadAll(efile.Data)
		if err != nil {
			continue
		}

		err = os.WriteFile(path.Join(storeDir, storepath), attData, 0600)
		if err != nil {
			continue
		}

		key := "cid:" + efile.CID
		url := fmt.Sprintf("/_/%s-%s", mailid, efile.CID)
		htmlbody = strings.ReplaceAll(htmlbody, key, url)
	}
	msg.HTMLBody = htmlbody
	rawHtml := "raw-" + mailid + ".html"
	err = os.WriteFile(path.Join(storeDir, rawHtml), []byte(msg.HTMLBody), 0600)
	if err != nil {
		log.Println("save raw html fail", mailid, rawHtml, req.Device, err)
		c.AbortWithError(http.StatusInternalServerError, errors.New("save raw html fail "+err.Error()))
		return
	}

	//
	//
	tmpl, err := template.New("").Parse(loadTemplate(req.Device))
	if err != nil {
		log.Println("load device template fail", mailid, req.Device, err)
		c.AbortWithError(http.StatusInternalServerError, errors.New("load device template fail "+err.Error()))
		return
	}
	headless, _ := strconv.ParseBool(req.Headless)
	if req.Headless == "on" {
		headless = true
	}
	vals := map[string]any{
		"author":    req.Author,
		"watermark": req.Watermark,
		"mail":      mail,
		"rawhtml":   rawHtml,
		"headless":  headless,
	}

	outf, err := os.Create(path.Join(storeDir, mailid+".html"))
	if err != nil {
		log.Println("create output html fail", mailid, req.Device, err)
		c.AbortWithError(http.StatusInternalServerError, errors.New("load device template fail "+err.Error()))
		return
	}
	defer func() {
		outf.Close()
	}()

	err = tmpl.Execute(outf, vals)
	if err != nil {
		log.Println("render output html fail", mailid, req.Device, err)
		c.AbortWithError(http.StatusInternalServerError, errors.New("render output html fail "+err.Error()))
		return
	}

	if chromePath == "" {
		chromePath, _ = launcher.LookPath()
	}
	b := rod.New().ControlURL(launcher.New().Bin(chromePath).MustLaunch())
	defer func() {
		if b != nil {
			b.Close()
		}
	}()

	switch req.Device {
	case DeviceiPhone:
		b = b.DefaultDevice(devices.IPhone6or7or8Plus)
	case DeviceWeb:
		if req.WithHiDPI {
			b = b.DefaultDevice(devices.LaptopWithHiDPIScreen)
		} else {
			b = b.DefaultDevice(devices.LaptopWithMDPIScreen)
		}

	}

	if req.Waitload == 0 {
		req.Waitload = 60
	}

	uri := fmt.Sprintf("%s/_/%s.html", localServerAddr, mailid)
	page := b.MustConnect().MustPage(uri)
	if req.Waitload > 0 {
		page = page.Timeout(time.Duration(req.Waitload) * time.Second)
		err = page.WaitLoad()
		if err != nil {
			log.Println("wait load timeout", req.Waitload, "seconds", err)
			page = page.CancelTimeout()
		}
	} else {
		page = page.MustWaitLoad()
	}

	var screenshotConfig *proto.PageCaptureScreenshot
	if len(floatVps) == 4 {
		screenshotConfig = &proto.PageCaptureScreenshot{
			Clip: &proto.PageViewport{
				X:      floatVps[0],
				Y:      floatVps[1],
				Width:  floatVps[2],
				Height: floatVps[3],
				Scale:  req.Scale,
			},
		}
	}

	var data []byte
	var contentType string
	if req.Format == "pdf" {
		r, err := page.PDF(&proto.PagePrintToPDF{})
		if err != nil {
			log.Println("PDF fail", uri, req.Device, err)
			c.AbortWithError(http.StatusInternalServerError, errors.New("pdf fail "+err.Error()))
			return
		}
		data, _ = io.ReadAll(r)
		contentType = "application/pdf"
	} else {
		data, err = page.Screenshot(true, screenshotConfig)
		if err != nil {
			log.Println("screenshot fail", uri, req.Device, err)
			c.AbortWithError(http.StatusInternalServerError, errors.New("screenshot fail "+err.Error()))
			return
		}
		contentType = "image/png"
	}
	c.Data(http.StatusOK, contentType, data)
	page.Close()
}
