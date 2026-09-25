package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPSender struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPSender(host, port, user, pass, from string) *SMTPSender {
	return &SMTPSender{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

func (s *SMTPSender) SendVerificationEmail(ctx context.Context, emailAddress, token string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	
	verifyURL := fmt.Sprintf("http://localhost:3000/verify-email?token=%s", token)

	subject := "Subject: Verify your Dogfood Hackathon email\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`<html>
		<body>
			<h2>Welcome to Dogfood Hackathon!</h2>
			<p>Please verify your email by clicking the link below:</p>
			<p><a href="%s">%s</a></p>
			<br/>
			<p>Or manually enter this token: <strong>%s</strong></p>
		</body>
	</html>`, verifyURL, verifyURL, token)

	msg := []byte("To: " + emailAddress + "\r\n" + subject + mime + body)

	var auth smtp.Auth
	if s.user != "" && s.pass != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}

	return smtp.SendMail(addr, auth, s.from, []string{emailAddress}, msg)
}
