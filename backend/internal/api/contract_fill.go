package api

import (
	"html"
	"strings"

	"github.com/prontuario/backend/internal/repo"
)

func escapeHTML(s string) string { return html.EscapeString(s) }

func strPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// BuildGuardianSignatureHTML retorna o HTML da assinatura do responsável na fonte escolhida (cursive, brush, dancing).
func BuildGuardianSignatureHTML(guardianName, fontKey string) string {
	name := escapeHTML(guardianName)
	if name == "" {
		return ""
	}
	fontFamily := "'Brush Script MT', 'Segoe Script', 'Dancing Script', cursive"
	switch strings.ToLower(strings.TrimSpace(fontKey)) {
	case "brush":
		fontFamily = "'Brush Script MT', cursive"
	case "dancing":
		fontFamily = "'Dancing Script', cursive"
	case "cursive", "segoe":
		fontFamily = "'Segoe Script', cursive"
	}
	return `<span style="font-family: ` + fontFamily + `; font-size: 1.25em;">` + name + `</span>`
}

// FillContractBody substitui no body_html do modelo os placeholders pelos dados do paciente, responsável e contratado.
// Placeholders: [PACIENTE_NOME], [RESPONSAVEL_*], [CONTRATANTE], [CONTRATADO], [OBJETO], [TIPO_SERVICO], [PERIODICIDADE],
// [VALOR], [ASSINATURA_PROFISSIONAL], [ASSINATURA_RESPONSAVEL], [DATA_INICIO], [DATA_FIM], [CONSULTAS_PREVISTAS], [DATA].
// O local deve estar escrito no template (não há substituição de [LOCAL]). dataAssinatura: para [DATA] (ex.: "11/02/2025").
// guardianSignatureHTML: se vazio, [ASSINATURA_RESPONSAVEL] não é substituído; senão usa esse HTML.
// consultasPrevistas: texto com os horários pré-agendados.
// guardianAddress: endereço formatado do responsável (ex.: FormatAddressToLines(addr)); preenchido pelo chamador a partir de guardian.AddressID.
func FillContractBody(body string, patient *repo.Patient, guardian *repo.LegalGuardian, contratado, objeto string, tipoServico, periodicidade, valor string, signatureImageData *string, professionalName *string, dataInicio, dataFim string, guardianSignatureHTML string, consultasPrevistas string, local, dataAssinatura string, guardianAddress string) string {
	patientName := ""
	patientBirth := ""
	if patient != nil {
		patientName = patient.FullName
		if patient.BirthDate != nil {
			patientBirth = *patient.BirthDate
		}
	}
	guardianName := ""
	guardianAddr := guardianAddress
	guardianBirth := ""
	if guardian != nil {
		guardianName = guardian.FullName
		if guardian.BirthDate != nil {
			guardianBirth = *guardian.BirthDate
		}
	}
	guardianCPF := "conforme cadastro"
	if objeto == "" && tipoServico != "" {
		objeto = tipoServico
	}
	if objeto == "" {
		objeto = "prestação de serviços"
	}
	if tipoServico == "" {
		tipoServico = objeto
	}
	if periodicidade == "" {
		periodicidade = "não informada"
	}
	if valor == "" {
		valor = "não informado"
	}
	body = strings.ReplaceAll(body, "[PACIENTE_NOME]", patientName)
	body = strings.ReplaceAll(body, "[PACIENTE_NASCIMENTO]", patientBirth)
	body = strings.ReplaceAll(body, "[RESPONSAVEL_NOME]", guardianName)
	guardianEmail := ""
	if guardian != nil {
		guardianEmail = guardian.Email
	}
	body = strings.ReplaceAll(body, "[RESPONSAVEL_EMAIL]", guardianEmail)
	body = strings.ReplaceAll(body, "[RESPONSAVEL_ENDERECO]", guardianAddr)
	body = strings.ReplaceAll(body, "[RESPONSAVEL_NASCIMENTO]", guardianBirth)
	body = strings.ReplaceAll(body, "[RESPONSAVEL_CPF]", guardianCPF)
	body = strings.ReplaceAll(body, "[CONTRATANTE]", guardianName)
	body = strings.ReplaceAll(body, "[CONTRATADO]", contratado)
	body = strings.ReplaceAll(body, "[OBJETO]", objeto)
	body = strings.ReplaceAll(body, "[TIPO_SERVICO]", tipoServico)
	body = strings.ReplaceAll(body, "[PERIODICIDADE]", periodicidade)
	body = strings.ReplaceAll(body, "[VALOR]", valor)
	sigHTML := ""
	if signatureImageData != nil && *signatureImageData != "" {
		sigHTML = `<img src="` + *signatureImageData + `" alt="Assinatura do profissional" style="max-height:56px;max-width:200px;display:block;" />`
	} else {
		// Sempre exibir nome do profissional em letra cursiva quando não houver imagem
		profName := ""
		if professionalName != nil && *professionalName != "" {
			profName = *professionalName
		}
		if profName != "" {
			sigHTML = `<span style="font-family: 'Brush Script MT', 'Segoe Script', 'Dancing Script', cursive; font-size: 1.35em;">` + escapeHTML(profName) + `</span>`
		} else {
			sigHTML = `<span style="color:#9ca3af;">_________________</span>`
		}
	}
	body = strings.ReplaceAll(body, "[ASSINATURA_PROFISSIONAL]", sigHTML)
	if dataInicio == "" {
		dataInicio = "não informado"
	}
	if dataFim == "" {
		dataFim = "não há data de término prevista (vigente até nova alteração)"
	}
	body = strings.ReplaceAll(body, "[DATA_INICIO]", dataInicio)
	body = strings.ReplaceAll(body, "[DATA_FIM]", dataFim)
	if guardianSignatureHTML == "" {
		if guardianName != "" {
			guardianSignatureHTML = `<span style="font-family: 'Segoe Script', cursive; font-size: 1.25em;">` + escapeHTML(guardianName) + `</span>`
		} else {
			guardianSignatureHTML = `<span style="color:#9ca3af;">_________________</span>`
		}
	}
	body = strings.ReplaceAll(body, "[ASSINATURA_RESPONSAVEL]", guardianSignatureHTML)
	if consultasPrevistas == "" {
		consultasPrevistas = "Não há horários pré-agendados."
	}
	body = strings.ReplaceAll(body, "[CONSULTAS_PREVISTAS]", consultasPrevistas)
	// [LOCAL] não é mais substituído — o template já deve trazer o local escrito (ex.: Joinville).
	if dataAssinatura == "" {
		dataAssinatura = "___/___/______"
	}
	body = strings.ReplaceAll(body, "[DATA]", dataAssinatura)
	return body
}

// FormatScheduleRulesText formata as regras de agendamento para exibição no contrato (dia da semana + horário).
// dayOfWeek: 0=domingo, 1=segunda, ..., 6=sábado.
func FormatScheduleRulesText(rules []repo.ContractScheduleRule) string {
	if len(rules) == 0 {
		return ""
	}
	dayNames := []string{"Domingo", "Segunda-feira", "Terça-feira", "Quarta-feira", "Quinta-feira", "Sexta-feira", "Sábado"}
	var parts []string
	for _, r := range rules {
		if r.DayOfWeek >= 0 && r.DayOfWeek < 7 {
			t := r.SlotTime.Format("15:04")
			parts = append(parts, dayNames[r.DayOfWeek]+" às "+t)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Consultas previstas: " + strings.Join(parts, "; ") + "."
}
