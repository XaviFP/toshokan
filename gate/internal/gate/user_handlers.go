package gate

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	userPB "github.com/XaviFP/toshokan/user/api/proto/v1"
)

type tokenResponse struct {
	Token string `json:"token"`
}

func RegisterUserRoutes(r *gin.Engine, enableSignup bool, userClient userPB.UserAPIClient) {
	if enableSignup {
		r.POST("/signup", func(ctx *gin.Context) {
			signUp(ctx, userClient)
		})
	}

	r.POST("/login", func(ctx *gin.Context) {
		logIn(ctx, userClient)
	})
}

func signUp(ctx *gin.Context, userClient userPB.UserAPIClient) {
	var req userPB.SignUpRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Username) < 3 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "username must be at least 3 characters"})
		return
	}
	if len(req.Password) < 8 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	res, err := userClient.SignUp(ctx, &req)
	if err != nil {
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "user:") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, tokenResponse{res.Token})
}

func logIn(ctx *gin.Context, userClient userPB.UserAPIClient) {
	var req userPB.LogInRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := userClient.LogIn(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, tokenResponse{res.Token})
}
