package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail"
)

// this directive embeds the files in the templates directory into the binary

//go:embed templates
var templateFS embed.FS

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// creates a new mailer instance with the provided SMTP server configuration
func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	// 5 seconds timeout for sending emails
	dialer.Timeout = time.Second * 5

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// sends an email to the specified recipient using the provided template file
func (m Mailer) Send(to, templateFile string, data any) error {
	// parse the template file from the embedded filesystem using the ParseFS function.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the "subject" template and store the result in the subject variable.
	// the data will be passed to the template as a map[string]any, and will replace
	// the {{.}} placeholder in the template.
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// Follow the same pattern to execute the "plainBody" template and store the result
	// in the plainBody variable.
	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// new message instance
	msg := mail.NewMessage()
	// setting data for the message
	msg.SetHeader("To", to)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	// used to add an alternative body to the message. in this case, we are adding
	// the htmlBody as an alternative to the plain text body
	msg.AddAlternative("text/html", htmlBody.String())

	// this opens a connection to the SMTP server, sends the message, then
	// closes the connection. If there is a timeout, it will return a "dial
	// tcp: i/o timeout" error.
	for range 3 {
		err = m.dialer.DialAndSend(msg)
		if err != nil {
			time.Sleep(time.Millisecond * 500)
			continue
		}

		return nil
	}

	return err
}
