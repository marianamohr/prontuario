package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"
)

// SignatureBlock dados do carimbo de assinatura no PDF.
type SignatureBlock struct {
	SignerName                   string
	SignerEmail                  string
	SignedAt                     string
	PDFSHA256                    string
	VerificationToken            string
	VerificationURL              string
	ExplanatoryText              string
	ProfessionalSignatureDataURL *string // data URL (ex.: data:image/png;base64,...) para imagem da assinatura do profissional
	ProfessionalName             *string // nome do profissional (usado em fonte cursiva quando não há imagem)
	GuardianSignatureName        string  // nome do responsável para exibir como assinatura em cursiva no bloco
}

// decodeDataURLImage extrai tipo (png/jpeg) e bytes de um data URL (data:image/png;base64,...).
func decodeDataURLImage(dataURL string) (ext string, data []byte, ok bool) {
	dataURL = strings.TrimSpace(dataURL)
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, false
	}
	idx := strings.Index(dataURL, ";base64,")
	if idx < 0 {
		return "", nil, false
	}
	header := dataURL[5:idx] // "image/png" ou "image/jpeg"
	if strings.HasPrefix(header, "image/png") {
		ext = "png"
	} else if strings.HasPrefix(header, "image/jpeg") || strings.HasPrefix(header, "image/jpg") {
		ext = "jpeg"
	} else {
		return "", nil, false
	}
	b64 := dataURL[idx+8:]
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(data) == 0 {
		return "", nil, false
	}
	return ext, data, true
}

// BuildContractPDF gera PDF do contrato: bodyHTML como texto (simplificado) + bloco de assinatura com QR.
func BuildContractPDF(bodyText string, block SignatureBlock) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 6, bodyText, "", "", false)

	// Assinatura do profissional no final da primeira página: imagem ou nome em cursiva
	hasSigImage := block.ProfessionalSignatureDataURL != nil && *block.ProfessionalSignatureDataURL != ""
	if hasSigImage {
		if ext, imgData, ok := decodeDataURLImage(*block.ProfessionalSignatureDataURL); ok {
			alias := "profsig"
			if pdf.RegisterImageReader(alias, ext, bytes.NewReader(imgData)) != nil {
				pdf.Ln(4)
				pdf.Image(alias, 15, pdf.GetY(), 50, 18, false, "", 0, "")
				pdf.SetY(pdf.GetY() + 19)
			}
		}
	} else if block.ProfessionalName != nil && *block.ProfessionalName != "" {
		pdf.Ln(4)
		pdf.SetFont("Times", "I", 12)
		pdf.CellFormat(0, 8, *block.ProfessionalName, "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Assinatura Eletronica", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(4)
	if block.GuardianSignatureName != "" {
		pdf.SetFont("Times", "I", 12)
		pdf.CellFormat(0, 8, block.GuardianSignatureName, "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
	}
	pdf.CellFormat(0, 6, "Nome do assinante: "+block.SignerName, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "E-mail: "+block.SignerEmail, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Data/hora: "+block.SignedAt, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Hash SHA-256 do documento: "+block.PDFSHA256, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Token de verificacao: "+block.VerificationToken, "", 1, "L", false, 0, "")
	pdf.Ln(4)
	if block.VerificationURL != "" {
		qrPNG, err := qrcode.Encode(block.VerificationURL, qrcode.Medium, 128)
		if err == nil {
			tmpFile, err := os.CreateTemp("", "qr-*.png")
			if err == nil {
				tmpFile.Write(qrPNG)
				path := tmpFile.Name()
				tmpFile.Close()
				defer os.Remove(path)
				pdf.RegisterImage(path, "PNG")
				pdf.Image(path, 15, pdf.GetY(), 30, 30, false, "", 0, "")
				pdf.SetY(pdf.GetY() + 32)
			}
		}
		pdf.CellFormat(0, 6, "Link para verificacao: "+block.VerificationURL, "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)
	expl := block.ExplanatoryText
	if expl == "" {
		expl = "Este documento foi assinado eletronicamente. A autenticidade pode ser verificada pelo link e hash acima. Nao utiliza certificado digital ICP-Brasil."
	}
	pdf.MultiCell(0, 5, expl, "", "", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BodyFromHTML muito simplificado: remove tags para texto plano no PDF.
func BodyFromHTML(html string) string {
	// Coloca quebras onde havia blocos
	var out []byte
	inTag := false
	for i := 0; i < len(html); i++ {
		c := html[i]
		if c == '<' {
			inTag = true
			continue
		}
		if inTag {
			if c == '>' {
				inTag = false
				out = append(out, ' ')
			}
			continue
		}
		if c == '&' {
			// entity simplificada
			if i+4 <= len(html) && html[i:i+4] == "&lt;" {
				out = append(out, '<')
				i += 3
				continue
			}
			if i+4 <= len(html) && html[i:i+4] == "&gt;" {
				out = append(out, '>')
				i += 3
				continue
			}
			if i+5 <= len(html) && html[i:i+5] == "&amp;" {
				out = append(out, '&')
				i += 4
				continue
			}
		}
		out = append(out, c)
	}
	return string(out)
}

func FormatSignatureBlock(signerName, signerEmail, signedAt, pdfSHA256, verificationToken, appPublicURL string) SignatureBlock {
	verURL := ""
	if appPublicURL != "" && verificationToken != "" {
		verURL = fmt.Sprintf("%s/verify/%s", appPublicURL, verificationToken)
	}
	return SignatureBlock{
		SignerName:        signerName,
		SignerEmail:       signerEmail,
		SignedAt:          signedAt,
		PDFSHA256:         pdfSHA256,
		VerificationToken: verificationToken,
		VerificationURL:   verURL,
	}
}

// WritePDFTo escreve o PDF no writer (para resposta HTTP ou arquivo).
func WritePDFTo(bodyText string, block SignatureBlock, w io.Writer) error {
	b, err := BuildContractPDF(bodyText, block)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
