package analysis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct {
	analysisService AnalysisService
}

func NewAnalysisHandler(analysisService AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{analysisService: analysisService}
}

type AnalyzeRequest struct {
	Text     string `json:"text" binding:"required"`
	Language string `json:"language" binding:"required,len=2"`
}

func (h *AnalysisHandler) AnalyzeText(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var userID uint
	switch v := userIDInterface.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.analysisService.AnalyzeText(c.Request.Context(), userID, req.Language, req.Text)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
