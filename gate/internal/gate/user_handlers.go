package gate

import (
	"net/http"

	"github.com/gin-gonic/gin"

	userPB "github.com/XaviFP/toshokan/user/api/proto/v1"
)

type tokenResponse struct {
	Token string `json:"token"`
}

func RegisterUserRoutes(r *gin.Engine, userClient userPB.UserAPIClient) {
	r.POST("/signup", func(ctx *gin.Context) {
		signUp(ctx, userClient)
	})

	r.POST("/login", func(ctx *gin.Context) {
		logIn(ctx, userClient)
	})
}

func signUp(ctx *gin.Context, userClient userPB.UserAPIClient) {
	var req userPB.SignUpRequest

	if err := ctx.BindJSON(&req); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, err.Error())
		return
	}

	res, err := userClient.SignUp(ctx, &req)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.IndentedJSON(http.StatusOK, tokenResponse{res.Token})
}

func logIn(ctx *gin.Context, userClient userPB.UserAPIClient) {
	var req userPB.LogInRequest

	if err := ctx.BindJSON(&req); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, err.Error())
		return
	}

	res, err := userClient.LogIn(ctx, &req)
	if err != nil {
		ctx.IndentedJSON(http.StatusUnauthorized, err.Error())
		return
	}

	ctx.IndentedJSON(http.StatusOK, tokenResponse{res.Token})
}
