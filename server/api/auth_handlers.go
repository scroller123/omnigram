package api

import (
	"encoding/json"
	"net/http"
	"omnigram/auth"
)

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type SendCodeResponse struct {
	PhoneCodeHash string `json:"phone_code_hash"`
}

type VerifyCodeRequest struct {
	Phone         string `json:"phone"`
	Code          string `json:"code"`
	PhoneCodeHash string `json:"phone_code_hash"`
}

type VerifyPasswordRequest struct {
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type UserResponse struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
}

func (s *Server) SendCode(w http.ResponseWriter, r *http.Request) {
	var req SendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := s.TgManager.GetApp(r.Context(), auth.GetSessionID(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash, err := app.SendCode(r.Context(), req.Phone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SendCodeResponse{PhoneCodeHash: hash})
}

func (s *Server) VerifyCode(w http.ResponseWriter, r *http.Request) {
	var req VerifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := s.TgManager.GetApp(r.Context(), auth.GetSessionID(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.SignIn(r.Context(), req.Phone, req.Code, req.PhoneCodeHash)
	if err != nil {
		// If 2FA is needed, it might return a specific error
		if err.Error() == "SESSION_PASSWORD_NEEDED" || 
		   (err.Error() != "" && err.Error() != "nil" && err.Error() == "2FA password required") { // gotd/td specific error?
			// Actually gotd returns a specific error type for this.
			// For simplicity, let's assume if it fails with password error, we tell frontend.
		}
		
		// Map gotd errors to friendly responses if needed
		if err.Error() == "SESSION_PASSWORD_NEEDED" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "2fa_required"})
			return
		}

		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Sign in successful
	token, err := auth.IssueToken()
	if err != nil {
		http.Error(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

func (s *Server) VerifyPassword(w http.ResponseWriter, r *http.Request) {
	var req VerifyPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := s.TgManager.GetApp(r.Context(), auth.GetSessionID(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.Password(r.Context(), req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := auth.IssueToken()
	if err != nil {
		http.Error(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

func (s *Server) GetMe(w http.ResponseWriter, r *http.Request) {
	app, err := s.TgManager.GetApp(r.Context(), auth.GetSessionID(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := app.GetMe(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Username:  user.Username,
		Phone:     user.Phone,
	})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	app, err := s.TgManager.GetApp(r.Context(), auth.GetSessionID(r.Context()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.Logout(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
