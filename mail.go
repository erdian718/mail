package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 配置
type Config struct {
	Addr    string
	Subject string
	From    string
	To      []string
	Bcc     []string
	Cc      []string
	Auth    smtp.Auth
}

// 邮件
type Mail struct {
	config  *Config
	buffer  *bytes.Buffer
	mwriter *multipart.Writer
}

// 新建邮件
func New(config *Config) *Mail {
	buffer := bytes.NewBuffer(nil)
	mwriter := multipart.NewWriter(buffer)

	buffer.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	buffer.WriteString("From: " + mime.BEncoding.Encode("utf-8", config.From) + "\r\n")
	buffer.WriteString("To: " + mime.BEncoding.Encode("utf-8", strings.Join(config.To, ", ")) + "\r\n")
	buffer.WriteString("Bcc: " + mime.BEncoding.Encode("utf-8", strings.Join(config.Bcc, ", ")) + "\r\n")
	buffer.WriteString("Cc: " + mime.BEncoding.Encode("utf-8", strings.Join(config.Cc, ", ")) + "\r\n")
	buffer.WriteString("Subject: " + mime.BEncoding.Encode("utf-8", config.Subject) + "\r\n")
	buffer.WriteString("Content-Type: multipart/mixed; boundary=" + mwriter.Boundary() + "\r\n")
	buffer.WriteString("MIME-Version: 1.0\r\n\r\n")

	return &Mail{
		config:  config,
		buffer:  buffer,
		mwriter: mwriter,
	}
}

// 发送
func (self *Mail) Send() error {
	if err := self.mwriter.Close(); err != nil {
		return err
	}

	c, err := smtp.Dial(self.config.Addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		host, _, _ := net.SplitHostPort(self.config.Addr)
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}

	if ok, _ := c.Extension("AUTH"); ok {
		if err := c.Auth(self.config.Auth); err != nil {
			return err
		}
	}

	if err := c.Mail(self.config.From); err != nil {
		return err
	}

	for _, addr := range self.config.To {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	for _, addr := range self.config.Cc {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	for _, addr := range self.config.Bcc {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = self.buffer.WriteTo(w)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}

// Part
func (self *Mail) Part(header textproto.MIMEHeader, f func(io.Writer) error) (err error) {
	var w io.Writer

	header.Set("Content-Transfer-Encoding", "base64")
	w, err = self.mwriter.CreatePart(header)
	if err != nil {
		return
	}

	bw := base64.NewEncoder(base64.StdEncoding, newWriter(w))
	err = f(bw)
	if err == nil {
		err = bw.Close()
	} else {
		bw.Close()
	}
	return
}

// 二进制Part
func (self *Mail) BinPart(name string, f func(io.Writer) error) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Disposition", "attachment; filename="+mime.BEncoding.Encode("utf-8", name))
	return self.Part(header, f)
}

// 文本Part
func (self *Mail) TextPart(typ string, f func(io.Writer) error) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", typ+"; charset=utf-8")
	return self.Part(header, f)
}

// 文本
func (self *Mail) Text(s string) error {
	return self.TextPart("text/plain", func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// 超文本
func (self *Mail) HTML(s string) error {
	return self.TextPart("text/html", func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// 添加附件(读端)
func (self *Mail) AttachReader(name string, r io.Reader) error {
	return self.BinPart(name, func(w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	})
}

// 添加附件(文件)
func (self *Mail) AttachFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return self.AttachReader(filepath.Base(path), f)
}
