package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"github.com/prontuario/backend/internal/crypto"
	"github.com/prontuario/backend/internal/pdf"
	"github.com/prontuario/backend/internal/repo"
)

type ContractByTokenResponse struct {
	ContractID      string `json:"contract_id"`
	PatientName     string `json:"patient_name"`
	GuardianName    string `json:"guardian_name"`
	BodyHTML        string `json:"body_html"`
	SignerRelation  string `json:"signer_relation"`
	SignerIsPatient bool   `json:"signer_is_patient"`
	Status          string `json:"status"`
	ClinicName      string `json:"clinic_name"` // nome da clínica que enviou o contrato (para o header da tela de assinatura)
}

func (h *Handler) GetContractByToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	c, tpl, patient, guardian, err := repo.ContractByAccessToken(r.Context(), h.Pool, token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusNotFound)
		return
	}
	if c.Status == "SIGNED" {
		http.Error(w, `{"error":"contract already signed"}`, http.StatusBadRequest)
		return
	}
	contratado := ""
	if clinic, err := repo.ClinicByID(r.Context(), h.Pool, c.ClinicID); err == nil {
		contratado = clinic.Name
	}
	var signatureData *string
	var professionalName *string
	if c.ProfessionalID != nil {
		if prof, err := repo.ProfessionalByID(r.Context(), h.Pool, *c.ProfessionalID); err == nil {
			signatureData = prof.SignatureImageData
			professionalName = &prof.FullName
		}
	}
	dataInicio := ""
	if c.StartDate != nil {
		dataInicio = c.StartDate.Format("02/01/2006")
	}
	dataFim := ""
	if c.EndDate != nil {
		dataFim = c.EndDate.Format("02/01/2006")
	}
	objeto := strPtrVal(tpl.TipoServico)
	if objeto == "" {
		objeto = tpl.Name
	}
	periodicidadeVal := strPtrVal(c.Periodicidade)
	if periodicidadeVal == "" {
		periodicidadeVal = strPtrVal(tpl.Periodicidade)
	}
	rules, errRules := repo.ListContractScheduleRules(r.Context(), h.Pool, c.ID)
	_ = errRules
	consultasPrevistas := FormatScheduleRulesText(rules)
	// Apenas [DATA] é substituída (data do dia em que a pessoa abre o link, DD/MM/AAAA). Local já vem no template.
	dataVal := time.Now().Format("02/01/2006")
	guardianAddrStr := FormatGuardianAddressForContract(r.Context(), h.Pool, guardian)
	bodyHTML := FillContractBody(tpl.BodyHTML, patient, guardian, contratado, objeto, strPtrVal(tpl.TipoServico), periodicidadeVal, strPtrVal(c.Valor), signatureData, professionalName, dataInicio, dataFim, "", consultasPrevistas, "", dataVal, guardianAddrStr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ContractByTokenResponse{
		ContractID:      c.ID.String(),
		PatientName:     patient.FullName,
		GuardianName:    guardian.FullName,
		BodyHTML:        bodyHTML,
		SignerRelation:  c.SignerRelation,
		SignerIsPatient: c.SignerIsPatient,
		Status:          c.Status,
		ClinicName:      contratado,
	})
}

type SignContractRequest struct {
	Token         string `json:"token"`
	AcceptedTerms bool   `json:"accepted_terms"`
	SignatureFont string `json:"signature_font"` // opcional: "cursive", "brush", "dancing" — fonte da assinatura do responsável
}

func (h *Handler) SignContract(w http.ResponseWriter, r *http.Request) {
	var req SignContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Token == "" || !req.AcceptedTerms {
		http.Error(w, `{"error":"token and accepted_terms true required"}`, http.StatusBadRequest)
		return
	}
	c, tpl, patient, guardian, err := repo.ContractByAccessToken(r.Context(), h.Pool, req.Token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
		return
	}
	if c.Status == "SIGNED" {
		http.Error(w, `{"error":"contract already signed"}`, http.StatusBadRequest)
		return
	}
	guardianID := auth.UserIDFrom(r.Context())
	if guardianID != "" {
		if gUUID, e := uuid.Parse(guardianID); e == nil && gUUID != guardian.ID {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
	}
	contratado := ""
	if clinic, err := repo.ClinicByID(r.Context(), h.Pool, c.ClinicID); err == nil {
		contratado = clinic.Name
	}
	var signatureData *string
	var professionalName *string
	if c.ProfessionalID != nil {
		if prof, err := repo.ProfessionalByID(r.Context(), h.Pool, *c.ProfessionalID); err == nil {
			signatureData = prof.SignatureImageData
			professionalName = &prof.FullName
		}
	}
	dataInicio := ""
	if c.StartDate != nil {
		dataInicio = c.StartDate.Format("02/01/2006")
	}
	dataFim := ""
	if c.EndDate != nil {
		dataFim = c.EndDate.Format("02/01/2006")
	}
	guardianSigHTML := BuildGuardianSignatureHTML(guardian.FullName, req.SignatureFont)
	objeto := strPtrVal(tpl.TipoServico)
	if objeto == "" {
		objeto = tpl.Name
	}
	periodicidadeVal := strPtrVal(c.Periodicidade)
	if periodicidadeVal == "" {
		periodicidadeVal = strPtrVal(tpl.Periodicidade)
	}
	rules, errRules := repo.ListContractScheduleRules(r.Context(), h.Pool, c.ID)
	_ = errRules
	consultasPrevistas := FormatScheduleRulesText(rules)
	// Data/hora real da assinatura (para o bloco de assinatura eletrônica no PDF)
	signedAtReal := time.Now().Format("02/01/2006 15:04:05")
	// Apenas [DATA] no corpo: data do dia da assinatura (DD/MM/AAAA). Local já vem no template.
	dataNoCorpo := time.Now().Format("02/01/2006")
	guardianAddrStr := FormatGuardianAddressForContract(r.Context(), h.Pool, guardian)
	bodyHTML := FillContractBody(tpl.BodyHTML, patient, guardian, contratado, objeto, strPtrVal(tpl.TipoServico), periodicidadeVal, strPtrVal(c.Valor), signatureData, professionalName, dataInicio, dataFim, guardianSigHTML, consultasPrevistas, "", dataNoCorpo, guardianAddrStr)
	verificationToken := uuid.New().String()
	bodyText := pdf.BodyFromHTML(bodyHTML)
	block := pdf.FormatSignatureBlock(guardian.FullName, guardian.Email, signedAtReal, "", verificationToken, h.Cfg.AppPublicURL)
	block.ProfessionalSignatureDataURL = signatureData
	block.ProfessionalName = professionalName
	block.GuardianSignatureName = guardian.FullName
	pdfBytes, err := pdf.BuildContractPDF(bodyText, block)
	if err != nil {
		http.Error(w, `{"error":"pdf generation"}`, http.StatusInternalServerError)
		return
	}
	pdfSHA256 := crypto.SHA256Hex(pdfBytes)
	block.PDFSHA256 = pdfSHA256
	var errPDF error
	pdfBytes, errPDF = pdf.BuildContractPDF(bodyText, block)
	_ = errPDF

	googleSubVal := ""
	if guardian.GoogleSub != nil {
		googleSubVal = *guardian.GoogleSub
	}
	auditFinal, errMarshal := json.Marshal(map[string]interface{}{
		"guardian_id":       guardian.ID.String(),
		"guardian_email":    guardian.Email,
		"google_sub":        googleSubVal,
		"ip":                r.RemoteAddr,
		"user_agent":        r.UserAgent(),
		"accepted_terms":    true,
		"accepted_terms_at": time.Now().Format(time.RFC3339),
		"signer_relation":   c.SignerRelation,
		"patient_id":        c.PatientID.String(),
		"legal_guardian_id": c.LegalGuardianID.String(),
		"template_version":  c.TemplateVersion,
		"pdf_sha256":        pdfSHA256,
	})
	if errMarshal != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	pdfURL := ""
	if h.Cfg.AppPublicURL != "" {
		pdfURL = h.Cfg.AppPublicURL + "/verify/" + verificationToken
	}
	_ = repo.SignContract(r.Context(), h.Pool, c.ID, pdfSHA256, verificationToken, auditFinal)
	_, _ = h.Pool.Exec(r.Context(), "UPDATE contracts SET pdf_url = $1 WHERE id = $2", pdfURL, c.ID)
	_ = repo.CancelOtherPendingContractsForPatientAndGuardian(r.Context(), h.Pool, c.ID, c.PatientID, c.LegalGuardianID)
	_ = repo.MarkContractAccessTokenUsed(r.Context(), h.Pool, req.Token)
	// Cria os compromissos na agenda a partir das regras do contrato (confirmados pela assinatura)
	if c.ProfessionalID != nil {
		startDate := time.Now()
		if c.StartDate != nil {
			startDate = *c.StartDate
		}
		endDate := startDate.AddDate(1, 0, 0)
		if c.EndDate != nil && c.EndDate.After(startDate) {
			endDate = *c.EndDate
		}
		maxAppointments := 0
		if c.NumAppointments != nil && *c.NumAppointments > 0 {
			maxAppointments = *c.NumAppointments
			// Garantir período mínimo para criar N compromissos (ex.: 1 regra = 1 por semana, então precisamos de N semanas)
			minEnd := startDate.AddDate(0, 0, maxAppointments*7)
			if endDate.Before(minEnd) {
				endDate = minEnd
			}
		}
		_ = repo.CreateAppointmentsFromContractRules(r.Context(), h.Pool, c.ID, c.ClinicID, *c.ProfessionalID, c.PatientID, startDate, endDate, 50, maxAppointments)
	}

	guardianUUID := guardian.ID
	_ = repo.CreateAuditEvent(r.Context(), h.Pool, "CONTRACT_SIGNED", "LEGAL_GUARDIAN", &guardianUUID, map[string]string{"contract_id": c.ID.String(), "patient_id": c.PatientID.String()})
	if h.sendContractSignedEmail != nil {
		_ = h.sendContractSignedEmail(guardian.Email, guardian.FullName, pdfBytes, verificationToken)
		_ = repo.CreateAuditEvent(r.Context(), h.Pool, "CONTRACT_SIGNED_EMAIL_SENT", "SYSTEM", nil, map[string]string{"contract_id": c.ID.String(), "to": guardian.Email})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":            "Contrato assinado com sucesso.",
		"verification_token": verificationToken,
	})
}
