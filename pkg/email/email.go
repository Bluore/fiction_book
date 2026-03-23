package email

import (
	"crypto/rand"
	"fiction-book/pkg/config"
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"
)

func sendEmail(apiKey string, context *resend.SendEmailRequest) error {
	client := resend.NewClient(apiKey)

	params := context

	sent, err := client.Emails.Send(params)
	if err != nil {
		log.Printf("Error sending email: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("email sent: %s", sent.Id)
	return nil
}

func CreateEmail(cfg *config.Config, receiver []string, subject string, content string) error {
	return sendEmail(cfg.Resend.ApiKey, &resend.SendEmailRequest{
		From:    cfg.Resend.From,
		To:      receiver,
		Subject: subject,
		Html:    content,
	})
}

func CreateEmailOne(cfg *config.Config, receiver string, subject string, content string) error {
	return CreateEmail(cfg, []string{receiver}, subject, content)
}

func CreateVerifyNumber() (string, error) {
	var randb [6]byte
	if _, err := rand.Read(randb[:]); err != nil {
		return "", err
	}
	num := ""
	for _, b := range randb {
		num += fmt.Sprintf("%d", b%10)
	}
	return num, nil
}
