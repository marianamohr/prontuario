package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strconv"
	"text/template"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Pass     string
	FromName string
	FromAddr string
}

func (c *Config) Send(to, subject, body string, html bool) error {
	port := c.Port
	if port == 0 {
		port = 25
	}
	addr := fmt.Sprintf("%s:%d", c.Host, port)
	from := c.FromAddr
	if c.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.FromName, c.FromAddr)
	}
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"Content-Type": "text/plain; charset=UTF-8",
	}
	if html {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	}
	var buf bytes.Buffer
	for k, v := range headers {
		buf.WriteString(k + ": " + v + "\r\n")
	}
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return smtp.SendMail(addr, c.authForSend(), c.FromAddr, []string{to}, buf.Bytes())
}

// authForSend returns nil when User is empty (e.g. MailHog), so no AUTH is sent.
func (c *Config) authForSend() smtp.Auth {
	if c.User != "" {
		return smtp.PlainAuth("", c.User, c.Pass, c.Host)
	}
	return nil
}

func (c *Config) SendPasswordReset(to, resetURL string) error {
	tpl := `Olá,

Você solicitou a redefinição de senha. Clique no link abaixo (válido por 1 hora):

{{.ResetURL}}

Se você não solicitou isso, ignore este e-mail.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"ResetURL": resetURL}); err != nil {
		return err
	}
	return c.Send(to, "Redefinição de senha - Prontuário Saúde", b.String(), false)
}

func (c *Config) SendContractToSign(to, fullName, signURL string) error {
	tpl := `Olá, {{.FullName}},

Há um contrato disponível para sua assinatura. Acesse o link abaixo para ler e assinar (válido por 7 dias):

{{.SignURL}}

Se você não esperava este e-mail, ignore-o.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"FullName": fullName, "SignURL": signURL}); err != nil {
		return err
	}
	return c.Send(to, "Contrato para assinatura - Prontuário Saúde", b.String(), false)
}

// SendContractCancelled envia e-mail ao responsável informando que o contrato foi cancelado (tornado ineligível).
func (c *Config) SendContractCancelled(to, fullName string) error {
	tpl := `Olá, {{.FullName}},

Informamos que o contrato que estava em seu nome foi cancelado e está inativo (tornado ineligível).

Se você tiver dúvidas, entre em contato com a clínica ou o profissional que atende.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"FullName": fullName}); err != nil {
		return err
	}
	return c.Send(to, "Contrato cancelado - Prontuário Saúde", b.String(), false)
}

// SendContractEnded envia e-mail ao responsável informando que o contrato foi encerrado (serviço prestado até a data).
func (c *Config) SendContractEnded(to, fullName, endDate string) error {
	tpl := `Olá, {{.FullName}},

Informamos que o contrato que estava em seu nome foi encerrado. O serviço foi prestado até {{.EndDate}} e, a partir dessa data, não é mais ofertado.

Se você tiver dúvidas, entre em contato com a clínica ou o profissional que atende.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"FullName": fullName, "EndDate": endDate}); err != nil {
		return err
	}
	return c.Send(to, "Contrato encerrado - Prontuário Saúde", b.String(), false)
}

func (c *Config) SendInvite(to, fullName, registerURL string) error {
	tpl := `Olá, {{.FullName}},

Você foi convidado a se cadastrar como profissional no Prontuário Saúde. Para concluir seu cadastro (dados complementares e senha), acesse o link abaixo:

{{.RegisterURL}}

Este link expira em 7 dias. Se você não esperava este convite, ignore este e-mail.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"FullName": fullName, "RegisterURL": registerURL}); err != nil {
		return err
	}
	return c.Send(to, "Convite para cadastro - Prontuário Saúde", b.String(), false)
}

// SendPatientInvite envia um e-mail ao responsável legal para completar cadastro do paciente via link.
func (c *Config) SendPatientInvite(to, fullName, registerURL string) error {
	tpl := `Olá, {{.FullName}},

Você recebeu um link para completar o cadastro do paciente no Prontuário Saúde (CPF, endereço e datas).

Para continuar, acesse:

{{.RegisterURL}}

Este link expira em 7 dias. Se você não esperava este e-mail, ignore.`
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]string{"FullName": fullName, "RegisterURL": registerURL}); err != nil {
		return err
	}
	return c.Send(to, "Convite para cadastro de paciente - Prontuário Saúde", b.String(), false)
}

func PortFromString(s string) int {
	n, err := strconv.Atoi(s)
	_ = err
	return n
}

func (c *Config) SendWithAttachment(to, subject, body string, attachmentName string, attachmentPDF []byte) error {
	port := c.Port
	if port == 0 {
		port = 25
	}
	addr := fmt.Sprintf("%s:%d", c.Host, port)
	from := c.FromAddr
	if c.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.FromName, c.FromAddr)
	}
	boundary := "boundary-prontuario-pdf"
	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n--" + boundary + "\r\n")
	buf.WriteString("Content-Type: application/pdf; name=\"" + attachmentName + "\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("Content-Disposition: attachment; filename=\"" + attachmentName + "\"\r\n\r\n")
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)
	_, _ = b64.Write(attachmentPDF)
	_ = b64.Close()
	buf.WriteString("\r\n--" + boundary + "--\r\n")
	return smtp.SendMail(addr, c.authForSend(), c.FromAddr, []string{to}, buf.Bytes())
}
