package api

import (
	"encoding/json"
	"log"
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
	// Data/hora real da assinatura no fuso do Brasil (para o bloco de assinatura eletrônica no PDF)
	locBR, errLoc := time.LoadLocation("America/Sao_Paulo")
	if errLoc != nil {
		locBR = time.UTC
	}
	nowBR := time.Now().In(locBR)
	signedAtReal := nowBR.Format("02/01/2006 15:04:05")
	// Apenas [DATA] no corpo: data do dia da assinatura (DD/MM/AAAA). Local já vem no template.
	dataNoCorpo := nowBR.Format("02/01/2006")
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
	// Atualiza compromissos PRE_AGENDADO para AGENDADO (criados no envio do contrato)
	_, _ = repo.UpdateAppointmentsStatusByContract(r.Context(), h.Pool, c.ID, "AGENDADO")

	guardianUUID := guardian.ID
	_ = repo.CreateAuditEvent(r.Context(), h.Pool, "CONTRACT_SIGNED", "LEGAL_GUARDIAN", &guardianUUID, map[string]string{"contract_id": c.ID.String(), "patient_id": c.PatientID.String()})
	if h.sendContractSignedEmail != nil {
		log.Printf("[email] sending signed contract (PDF) to %s", guardian.Email)
		if err := h.sendContractSignedEmail(guardian.Email, guardian.FullName, pdfBytes, verificationToken); err != nil {
			log.Printf("[email] failed to send signed contract to %s: %v", guardian.Email, err)
		} else {
			_ = repo.CreateAuditEvent(r.Context(), h.Pool, "CONTRACT_SIGNED_EMAIL_SENT", "SYSTEM", nil, map[string]string{"contract_id": c.ID.String(), "to": guardian.Email})
		}
	} else {
		log.Printf("[email] signed contract email disabled (would send to %s)", guardian.Email)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":            "Contract signed successfully.",
		"verification_token": verificationToken,
	})
}
