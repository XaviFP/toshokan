package gate

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbDealer "github.com/XaviFP/toshokan/dealer/api/proto/v1"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

const (
	DefaultReviewCards = 20
	MaxReviewCards     = 100
)

type reviewResponse struct {
	Cards []cardResponse `json:"cards"`
}

func GetReview(ctx *gin.Context, dealerClient pbDealer.DealerClient, decksClient pbDeck.DecksAPIClient) {
	maxCards := uint64(DefaultReviewCards)
	if param := ctx.Query("max_cards"); param != "" {
		parsedParam, err := strconv.ParseUint(param, 10, 32)
		if err != nil || parsedParam == 0 || parsedParam > MaxReviewCards {
			slog.Error("GetReview: failed to parse max_cards", "error", err, "maxCards", param)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("max_cards must be between 1 and %d", MaxReviewCards)})
			return
		}
		maxCards = parsedParam
	}

	deckID, hasDeckID := ctx.GetQuery("deck_id")
	courseID, hasCourseID := ctx.GetQuery("course_id")
	lessonID, hasLessonID := ctx.GetQuery("lesson_id")

	scopes := 0
	for _, has := range []bool{hasDeckID, hasCourseID, hasLessonID} {
		if has {
			scopes++
		}
	}
	if scopes > 1 {
		slog.Error("GetReview: more than one scope", "deckId", deckID, "courseId", courseID, "lessonId", lessonID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "set at most one of: deck_id, course_id, lesson_id"})
		return
	}

	req := &pbDealer.DealRequest{
		UserId:        getUserID(ctx),
		NumberOfCards: uint32(maxCards),
	}

	if hasDeckID {
		if _, err := uuid.Parse(deckID); err != nil {
			slog.Error("GetReview: failed to parse deck ID", "error", err, "deckId", deckID, "stack", errors.ErrorStack(err))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
			return
		}
		req.DeckIds = []string{deckID}
	}

	if hasCourseID {
		if _, err := uuid.Parse(courseID); err != nil {
			slog.Error("GetReview: failed to parse course ID", "error", err, "courseId", courseID, "stack", errors.ErrorStack(err))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id format"})
			return
		}
		req.CourseId = courseID
	}

	if hasLessonID {
		if _, err := uuid.Parse(lessonID); err != nil {
			slog.Error("GetReview: failed to parse lesson ID", "error", err, "lessonId", lessonID, "stack", errors.ErrorStack(err))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id format"})
			return
		}
		req.LessonId = lessonID
	}

	dealt, err := dealerClient.Deal(ctx, req)
	if err != nil {
		if status.Code(err) == codes.InvalidArgument {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": status.Convert(err).Message()})
			return
		}
		if status.Code(err) == codes.NotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			return
		}
		slog.Error("GetReview: Deal failed", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	out := reviewResponse{Cards: []cardResponse{}}
	if len(dealt.CardIds) == 0 {
		ctx.JSON(http.StatusOK, out)
		return
	}

	res, err := decksClient.GetCards(ctx, &pbDeck.GetCardsRequest{CardIds: dealt.CardIds})
	if err != nil {
		slog.Error("GetReview: GetCards failed", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	for _, id := range dealt.CardIds {
		if c, ok := res.Cards[id]; ok {
			out.Cards = append(out.Cards, toCardResponse(c))
		}
	}

	ctx.JSON(http.StatusOK, out)
}

func ReviewAnswers(ctx *gin.Context, dealerClient pbDealer.DealerClient) {
	var req struct {
		AnswerIDs []string `json:"answer_ids"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		slog.Error("ReviewAnswers: failed to bind JSON", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.AnswerIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "answer_ids must not be empty"})
		return
	}

	for _, id := range req.AnswerIDs {
		if _, err := uuid.Parse(id); err != nil {
			slog.Error("ReviewAnswers: failed to parse answer ID", "error", err, "answerId", id, "stack", errors.ErrorStack(err))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id format"})
			return
		}
	}

	_, err := dealerClient.ReviewAnswers(ctx, &pbDealer.ReviewAnswersRequest{
		UserId:    getUserID(ctx),
		AnswerIds: req.AnswerIDs,
	})
	if err != nil {
		// The answer IDs are already valid UUIDs, so dealer only rejects them when they don't exist
		// or the card isn't in the user's review pool.
		if status.Code(err) == codes.InvalidArgument {
			ctx.JSON(http.StatusNotFound, gin.H{"error": status.Convert(err).Message()})
			return
		}
		slog.Error("ReviewAnswers: ReviewAnswers failed", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func RegisterReviewRoutes(r *gin.RouterGroup, dealerClient pbDealer.DealerClient, decksClient pbDeck.DecksAPIClient) {
	r.GET("/review", func(ctx *gin.Context) {
		GetReview(ctx, dealerClient, decksClient)
	})

	r.POST("/review/answers", func(ctx *gin.Context) {
		ReviewAnswers(ctx, dealerClient)
	})
}
