package service

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"searchpix/internal/config"
)

type EmailSender struct {
	cfg config.EmailConfig
}

func NewEmailSender(cfg config.EmailConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

func (s *EmailSender) isGmailProvider() bool {
	return strings.EqualFold(strings.TrimSpace(s.cfg.Provider), "gmail")
}

func (s *EmailSender) IsConfigured() bool {
	if s.cfg.SMTPHost == "" || s.cfg.SMTPPort == "" || s.cfg.From == "" {
		return false
	}
	if s.isGmailProvider() {
		// Gmail SMTP exige autenticação (usuário + senha de app).
		return s.cfg.SMTPUser != "" && s.cfg.SMTPPassword != ""
	}
	return true
}

func (s *EmailSender) Send(to, subject, body string) error {
	if !s.IsConfigured() {
		log.Printf("Email não configurado. provider=%s destino=%s assunto=%q", s.cfg.Provider, to, subject)
		return nil
	}
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.cfg.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg.String()))
}
