package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prontuario/backend/internal/auth"
	"gorm.io/gorm"
	"github.com/prontuario/backend/internal/repo"
)

func (h *Handler) ListContractTemplates(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var list []repo.ContractTemplate
	var profID *uuid.UUID
	if auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if p, e := uuid.Parse(userID); e == nil {
			profID = &p
		}
		list, err = repo.ContractTemplatesByClinicAndProfessional(r.Context(), h.DB, cid, profID)
	} else {
		list, err = repo.ContractTemplatesByClinic(r.Context(), h.DB, cid)
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version int    `json:"version"`
	}
	out := make([]item, len(list))
	for i := range list {
		out[i] = item{ID: list[i].ID.String(), Name: list[i].Name, Version: list[i].Version}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"templates": out})
}

func (h *Handler) CreateContractTemplate(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req struct {
		Name          string `json:"name"`
		BodyHTML      string `json:"body_html"`
		TipoServico   string `json:"tipo_servico"`
		Periodicidade string `json:"periodicidade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name and body_html required"}`, http.StatusBadRequest)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var profID *uuid.UUID
	if auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if p, e := uuid.Parse(userID); e == nil {
			profID = &p
		}
	}
	id, err := repo.CreateContractTemplate(r.Context(), h.DB, cid, profID, req.Name, req.BodyHTML, req.TipoServico, req.Periodicidade)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
}

func (h *Handler) GetContractTemplate(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	tpl, err := repo.ContractTemplateByIDAndClinic(r.Context(), h.DB, id, cid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[GetContractTemplate] repo error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	tipoServico := ""
	if tpl.TipoServico != nil {
		tipoServico = *tpl.TipoServico
	}
	periodicidade := ""
	if tpl.Periodicidade != nil {
		periodicidade = *tpl.Periodicidade
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id": tpl.ID.String(), "name": tpl.Name, "body_html": tpl.BodyHTML, "version": tpl.Version,
		"tipo_servico": tipoServico, "periodicidade": periodicidade,
	})
}

func (h *Handler) UpdateContractTemplate(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		Name          string `json:"name"`
		BodyHTML      string `json:"body_html"`
		TipoServico   string `json:"tipo_servico"`
		Periodicidade string `json:"periodicidade"`
		Version       int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if req.Version <= 0 {
		req.Version = 1
	}
	if err := repo.UpdateContractTemplate(r.Context(), h.DB, id, cid, req.Name, req.BodyHTML, req.TipoServico, req.Periodicidade, req.Version); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *Handler) DeleteContractTemplate(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if err := repo.DeleteContractTemplate(r.Context(), h.DB, id, cid); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			http.Error(w, `{"error":"Não é possível excluir: existem contratos vinculados a este modelo."}`, http.StatusConflict)
			return
		}
		log.Printf("[DeleteContractTemplate] %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListContracts(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	limit, offset := ParseLimitOffset(r)
	list, total, err := repo.ContractsByClinicPaginated(r.Context(), h.DB, cid, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		ID              string  `json:"id"`
		PatientID       string  `json:"patient_id"`
		LegalGuardianID string  `json:"legal_guardian_id"`
		SignerRelation  string  `json:"signer_relation"`
		SignerIsPatient bool    `json:"signer_is_patient"`
		Status          string  `json:"status"`
		SignedAt        *string `json:"signed_at,omitempty"`
	}
	out := make([]item, len(list))
	for i := range list {
		out[i] = item{
			ID: list[i].ID.String(), PatientID: list[i].PatientID.String(), LegalGuardianID: list[i].LegalGuardianID.String(),
			SignerRelation: list[i].SignerRelation, SignerIsPatient: list[i].SignerIsPatient, Status: list[i].Status,
		}
		if list[i].SignedAt != nil {
			s := list[i].SignedAt.Format(time.RFC3339)
			out[i].SignedAt = &s
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"contracts": out,
		"limit":     limit,
		"offset":    offset,
		"total":     total,
	})
}

// ListPendingContracts retorna os contratos PENDING da clínica (para a home: contratos que faltam assinar).
func (h *Handler) ListPendingContracts(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	list, err := repo.PendingContractsByClinic(r.Context(), h.DB, cid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		ID           string `json:"id"`
		PatientID    string `json:"patient_id"`
		PatientName  string `json:"patient_name"`
		TemplateName string `json:"template_name"`
		GuardianName string `json:"guardian_name"`
	}
	out := make([]item, len(list))
	for i := range list {
		out[i] = item{
			ID: list[i].ID.String(), PatientID: list[i].PatientID.String(),
			PatientName: list[i].PatientName, TemplateName: list[i].TemplateName, GuardianName: list[i].GuardianName,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"contracts": out})
}

// ListContractsForAgenda retorna apenas contratos assinados com nome do paciente e do modelo (para o modal de criar agendamentos na Agenda).
func (h *Handler) ListContractsForAgenda(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	list, err := repo.SignedContractsByClinicWithDetails(r.Context(), h.DB, cid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		ID           string `json:"id"`
		PatientID    string `json:"patient_id"`
		PatientName  string `json:"patient_name"`
		TemplateName string `json:"template_name"`
	}
	out := make([]item, len(list))
	for i := range list {
		out[i] = item{
			ID: list[i].ID.String(), PatientID: list[i].PatientID.String(),
			PatientName: list[i].PatientName, TemplateName: list[i].TemplateName,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"contracts": out})
}

func (h *Handler) CreateContract(w http.ResponseWriter, r *http.Request) {
	clinicID := auth.ClinicIDFrom(r.Context())
	if clinicID == nil || *clinicID == "" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if auth.RoleFrom(r.Context()) != auth.RoleProfessional && !auth.IsSuperAdmin(r.Context()) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req struct {
		PatientID       string `json:"patient_id"`
		LegalGuardianID string `json:"legal_guardian_id"`
		TemplateID      string `json:"template_id"`
		SignerRelation  string `json:"signer_relation"`
		SignerIsPatient bool   `json:"signer_is_patient"`
		DataInicio      string `json:"data_inicio"`   // opcional, YYYY-MM-DD
		DataFim         string `json:"data_fim"`      // opcional, YYYY-MM-DD
		Valor           string `json:"valor"`         // opcional, valor do serviço
		Periodicidade   string `json:"periodicidade"` // opcional
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.PatientID == "" || req.LegalGuardianID == "" || req.TemplateID == "" || req.SignerRelation == "" {
		http.Error(w, `{"error":"patient_id, legal_guardian_id, template_id, signer_relation required"}`, http.StatusBadRequest)
		return
	}
	cid, err := uuid.Parse(*clinicID)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		http.Error(w, `{"error":"invalid patient_id"}`, http.StatusBadRequest)
		return
	}
	guardianID, err := uuid.Parse(req.LegalGuardianID)
	if err != nil {
		http.Error(w, `{"error":"invalid legal_guardian_id"}`, http.StatusBadRequest)
		return
	}
	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		http.Error(w, `{"error":"invalid template_id"}`, http.StatusBadRequest)
		return
	}
	tpl, err := repo.ContractTemplateByIDAndClinic(r.Context(), h.DB, templateID, cid)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusBadRequest)
		return
	}
	_, err = repo.PatientByIDAndClinic(r.Context(), h.DB, patientID, cid)
	if err != nil {
		http.Error(w, `{"error":"patient not found"}`, http.StatusBadRequest)
		return
	}
	var profID *uuid.UUID
	if auth.RoleFrom(r.Context()) == auth.RoleProfessional {
		userID := auth.UserIDFrom(r.Context())
		if p, e := uuid.Parse(userID); e == nil {
			profID = &p
		}
	}
	var startDate, endDate *time.Time
	if req.DataInicio != "" {
		if t, err := time.Parse("2006-01-02", req.DataInicio); err == nil {
			startDate = &t
		}
	}
	if req.DataFim != "" {
		if t, err := time.Parse("2006-01-02", req.DataFim); err == nil {
			endDate = &t
		}
	}
	var valorPtr *string
	if req.Valor != "" {
		valorPtr = &req.Valor
	}
	var periodicidadePtr *string
	if req.Periodicidade != "" {
		periodicidadePtr = &req.Periodicidade
	}
	contractID, err := repo.CreateContract(r.Context(), h.DB, cid, patientID, guardianID, profID, templateID, req.SignerRelation, req.SignerIsPatient, tpl.Version, startDate, endDate, valorPtr, periodicidadePtr, nil, nil, nil)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	accessToken, err := repo.CreateContractAccessToken(r.Context(), h.DB, contractID, 7*24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	signURL := ""
	if h.Cfg.AppPublicURL != "" {
		signURL = h.Cfg.AppPublicURL + "/sign-contract?token=" + accessToken
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"contract_id":  contractID.String(),
		"access_token": accessToken,
		"sign_url":     signURL,
	})
}

func (h *Handler) GetContractVerify(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusNotFound)
		return
	}
	c, err := repo.ContractByVerificationToken(r.Context(), h.DB, token)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if c.Status != "SIGNED" {
		http.Error(w, `{"error":"contract not signed"}`, http.StatusBadRequest)
		return
	}
	signedAt := ""
	if c.SignedAt != nil {
		signedAt = c.SignedAt.Format(time.RFC3339)
	}
	bodyHTML := ""
	tpl, err := repo.ContractTemplateByID(r.Context(), h.DB, c.TemplateID)
	if err == nil {
		patient, errP := repo.PatientByID(r.Context(), h.DB, c.PatientID)
		_ = errP
		guardian, errG := repo.LegalGuardianByID(r.Context(), h.DB, c.LegalGuardianID)
		_ = errG
		contratado := ""
		if clinic, errClinic := repo.ClinicByID(r.Context(), h.DB, c.ClinicID); errClinic == nil && clinic != nil {
			contratado = clinic.Name
		}
		var sigData *string
		var profName *string
		if c.ProfessionalID != nil {
			if prof, errProf := repo.ProfessionalByID(r.Context(), h.DB, *c.ProfessionalID); errProf == nil && prof != nil {
				sigData = prof.SignatureImageData
				profName = &prof.FullName
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
		guardianSigHTML := ""
		if guardian != nil {
			guardianSigHTML = BuildGuardianSignatureHTML(guardian.FullName, "cursive")
		}
		objeto := strPtrVal(tpl.TipoServico)
		if objeto == "" {
			objeto = tpl.Name
		}
		periodicidadeVal := strPtrVal(c.Periodicidade)
		if periodicidadeVal == "" {
			periodicidadeVal = strPtrVal(tpl.Periodicidade)
		}
		rules, errRules := repo.ListContractScheduleRules(r.Context(), h.DB, c.ID)
		_ = errRules
		consultasPrevistas := FormatScheduleRulesText(rules)
		localVal := strPtrVal(c.SignPlace)
		dataAssinatura := ""
		if c.SignedAt != nil {
			dataAssinatura = c.SignedAt.Format("02/01/2006 15:04:05")
		}
		guardianAddrStr := FormatGuardianAddressForContract(r.Context(), h.DB, guardian)
		bodyHTML = FillContractBody(tpl.BodyHTML, patient, guardian, contratado, objeto, strPtrVal(tpl.TipoServico), periodicidadeVal, strPtrVal(c.Valor), sigData, profName, dataInicio, dataFim, guardianSigHTML, consultasPrevistas, localVal, dataAssinatura, guardianAddrStr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"contract_id": c.ID.String(), "status": c.Status, "signed_at": signedAt,
		"pdf_sha256": c.PDFSHA256, "verification_token": c.VerificationToken,
		"body_html": bodyHTML,
	})
}
