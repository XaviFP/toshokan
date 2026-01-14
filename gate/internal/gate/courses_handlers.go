package gate

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/XaviFP/toshokan/course/api/proto/v1"
)

func GetCourse(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	if courseID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id"})
		return
	}

	req := &pb.GetCourseRequest{CourseId: courseID}
	res, err := client.GetCourse(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("GetCourse: gRPC call failed", "error", err, "courseId", courseID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toCourseJSON(res.Course))
}

func GetLessons(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	if courseID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id"})
		return
	}

	pagination, err := parsePagination(ctx)
	if err != nil {
		slog.Error("GetLessons: failed to parse pagination", "error", err, "courseId", courseID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination parameters"})
		return
	}

	req := &pb.GetLessonsRequest{
		CourseId:   courseID,
		Pagination: pagination,
	}

	res, err := client.GetLessons(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("GetLessons: gRPC call failed", "error", err, "courseId", courseID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"edges":     convertLessonEdges(res.Lessons.Edges),
		"page_info": toPageInfoJSON(res.Lessons.PageInfo),
	})
}

func GetFocusedLessons(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	userID := getUserID(ctx)

	if courseID == "" || userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id or user id"})
		return
	}

	pagination, err := parsePagination(ctx)
	if err != nil {
		slog.Error("GetFocusedLessons: failed to parse pagination", "error", err, "courseId", courseID, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination parameters"})
		return
	}

	req := &pb.GetFocusedLessonsRequest{
		CourseId:   courseID,
		UserId:     userID,
		Pagination: pagination,
	}

	res, err := client.GetFocusedLessons(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("GetFocusedLessons: gRPC call failed", "error", err, "courseId", courseID, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"edges":     convertLessonWithProgressEdges(res.Lessons.Edges),
		"page_info": toPageInfoJSON(res.Lessons.PageInfo),
	})
}

func EnrollCourse(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	userID := getUserID(ctx)

	if courseID == "" || userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id or user id"})
		return
	}

	req := &pb.EnrollUserRequest{
		UserId:   userID,
		CourseId: courseID,
	}

	res, err := client.EnrollUser(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("EnrollCourse: gRPC call failed", "error", err, "courseId", courseID, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": res.Success})
}

func GetEnrolledCourses(ctx *gin.Context, client pb.CourseAPIClient) {
	userID := getUserID(ctx)

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing user id"})
		return
	}

	pagination, err := parsePagination(ctx)
	if err != nil {
		slog.Error("GetEnrolledCourses: failed to parse pagination", "error", err, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination parameters"})
		return
	}

	req := &pb.GetEnrolledCoursesRequest{
		UserId:     userID,
		Pagination: pagination,
	}

	res, err := client.GetEnrolledCourses(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("GetEnrolledCourses: gRPC call failed", "error", err, "userId", userID, "userIdLen", len(userID), "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"edges":     convertCourseWithProgressEdges(res.Courses.Edges),
		"page_info": toPageInfoJSON(res.Courses.PageInfo),
	})
}

func GetLessonState(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	lessonID := ctx.Param("lessonId")
	userID := getUserID(ctx)

	if courseID == "" || lessonID == "" || userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}

	req := &pb.GetLessonStateRequest{
		CourseId: courseID,
		LessonId: lessonID,
		UserId:   userID,
	}

	res, err := client.GetLessonState(ctx, req)
	if err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("GetLessonState: gRPC call failed", "error", err, "courseId", courseID, "lessonId", lessonID, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use protojson marshaler with EmitDefaultValues to include all fields
	marshaler := protojson.MarshalOptions{
		EmitDefaultValues: true,
		UseProtoNames:     true, // Use snake_case for JSON
	}

	jsonBytes, err := marshaler.Marshal(&pb.GetLessonStateResponse{LessonState: res.LessonState})
	if err != nil {
		slog.Error("GetLessonState: failed to marshal response", "error", err, "courseId", courseID, "lessonId", lessonID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal response"})
		return
	}

	ctx.Data(http.StatusOK, "application/json", jsonBytes)
}

func AnswerCards(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	lessonID := ctx.Param("lessonId")
	deckID := ctx.Param("deckId")
	userID := getUserID(ctx)

	if courseID == "" || lessonID == "" || deckID == "" || userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}

	var cardAnswers []*pb.CardAnswer
	if err := ctx.ShouldBindJSON(&cardAnswers); err != nil {
		slog.Error("AnswerCards: failed to bind JSON", "error", err, "courseId", courseID, "lessonId", lessonID, "deckId", deckID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := &pb.AnswerCardsRequest{
		UserId:      userID,
		CourseId:    courseID,
		LessonId:    lessonID,
		DeckId:      deckID,
		CardAnswers: cardAnswers,
	}

	res, err := client.AnswerCards(ctx, req)
	if err != nil {
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "lesson") || strings.Contains(err.Error(), "deck") || strings.Contains(err.Error(), "card") || strings.Contains(err.Error(), "invalid UUID") {
			slog.Error("AnswerCards: validation error", "error", err, "courseId", courseID, "lessonId", lessonID, "deckId", deckID, "userId", userID)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		slog.Error("AnswerCards: gRPC call failed", "error", err, "courseId", courseID, "lessonId", lessonID, "deckId", deckID, "userId", userID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func CreateCourse(ctx *gin.Context, client pb.CourseAPIClient) {
	var req struct {
		Order       int64  `json:"order"` // missing binding to allow zero
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createReq := &pb.CreateCourseRequest{
		Order:       req.Order,
		Title:       req.Title,
		Description: req.Description,
	}

	res, err := client.CreateCourse(ctx, createReq)
	if err != nil {
		slog.Error("CreateCourse: gRPC call failed", "error", err, "title", req.Title, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, toCourseJSON(res.Course))
}

func CreateLesson(ctx *gin.Context, client pb.CourseAPIClient) {
	courseID := ctx.Param("courseId")
	if courseID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id"})
		return
	}

	var req struct {
		Order       int64  `json:"order"` // missing binding to allow zero
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		Body        string `json:"body"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createReq := &pb.CreateLessonRequest{
		CourseId:    courseID,
		Order:       req.Order,
		Title:       req.Title,
		Description: req.Description,
		Body:        req.Body,
	}

	res, err := client.CreateLesson(ctx, createReq)
	if err != nil {
		//  TODO: Handle these errors properly
		if strings.Contains(err.Error(), "at least one deck in the body") {
			slog.Error("CreateLesson: validation error - no deck references", "error", err, "courseId", courseID, "title", req.Title)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("CreateLesson: gRPC call failed", "error", err, "courseId", courseID, "title", req.Title, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, toLessonJSON(res.Lesson))
}

func SyncState(ctx *gin.Context, client pb.CourseAPIClient) {
	userID := getUserID(ctx)

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing user id"})
		return
	}

	courseID := ctx.Param("courseId")
	if courseID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing course id"})
		return
	}

	if _, err := client.SyncState(ctx, &pb.SyncStateRequest{
		UserId:   userID,
		CourseId: courseID,
	}); err != nil {
		if isHandledError(ctx, err) {
			return
		}
		slog.Error("SyncState: gRPC call failed", "error", err, "userId", userID, "courseId", courseID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func RegisterCoursesRoutes(r *gin.RouterGroup, client pb.CourseAPIClient, adminCfg AdminConfig) {
	course := r.Group("/courses")
	{
		course.POST("", RequireAdmin(adminCfg, adminCfg.CreateCourseAdminOnly), func(ctx *gin.Context) {
			CreateCourse(ctx, client)
		})
		course.GET("/enrolled", func(ctx *gin.Context) {
			GetEnrolledCourses(ctx, client)
		})
		course.GET("/:courseId", func(ctx *gin.Context) {
			GetCourse(ctx, client)
		})
		course.POST("/:courseId/enroll", RequireAdmin(adminCfg, adminCfg.EnrollAdminOnly), func(ctx *gin.Context) {
			EnrollCourse(ctx, client)
		})
		course.GET("/:courseId/lessons", func(ctx *gin.Context) {
			GetLessons(ctx, client)
		})
		course.POST("/:courseId/lessons", RequireAdmin(adminCfg, adminCfg.CreateLessonAdminOnly), func(ctx *gin.Context) {
			CreateLesson(ctx, client)
		})
		course.GET("/:courseId/lessons/focused", func(ctx *gin.Context) {
			GetFocusedLessons(ctx, client)
		})
		course.GET("/:courseId/lessons/:lessonId/state", func(ctx *gin.Context) {
			GetLessonState(ctx, client)
		})
		course.POST("/:courseId/lessons/:lessonId/decks/:deckId/answer", func(ctx *gin.Context) {
			AnswerCards(ctx, client)
		})
		course.POST("/:courseId/sync", func(ctx *gin.Context) {
			SyncState(ctx, client)
		})
	}
}

// CourseJSON is a JSON-serializable version of pb.Course with timestamps as strings
type CourseJSON struct {
	ID          string  `json:"id"`
	Order       int64   `json:"order"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
	EditedAt    *string `json:"edited_at,omitempty"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

// LessonJSON is a JSON-serializable version of pb.Lesson with timestamps as strings
type LessonJSON struct {
	ID          string  `json:"id"`
	CourseID    string  `json:"course_id"`
	Order       int64   `json:"order"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Body        string  `json:"body"`
	CreatedAt   string  `json:"created_at"`
	EditedAt    *string `json:"edited_at,omitempty"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

// LessonEdgeJSON represents a lesson edge for JSON response
type LessonEdgeJSON struct {
	Node   LessonJSON `json:"node"`
	Cursor string     `json:"cursor"`
}

// LessonWithProgressJSON represents a lesson with progress for JSON response (flattened)
// The base lesson fields live at the top level alongside progress flags.
type LessonWithProgressJSON struct {
	LessonJSON
	IsCompleted bool `json:"is_completed"`
	IsCurrent   bool `json:"is_current"`
}

// LessonWithProgressEdgeJSON represents a lesson with progress edge for JSON response
type LessonWithProgressEdgeJSON struct {
	Node   LessonWithProgressJSON `json:"node"`
	Cursor string                 `json:"cursor"`
}

// CourseWithProgressJSON represents a course with progress for JSON response (flattened)
type CourseWithProgressJSON struct {
	CourseJSON
	CurrentLessonID string `json:"current_lesson_id"`
}

// CourseWithProgressEdgeJSON represents a course with progress edge for JSON response
type CourseWithProgressEdgeJSON struct {
	Node   CourseWithProgressJSON `json:"node"`
	Cursor string                 `json:"cursor"`
}

type PageInfoJSON struct {
	HasPreviousPage bool   `json:"has_previous_page"`
	HasNextPage     bool   `json:"has_next_page"`
	StartCursor     string `json:"start_cursor"`
	EndCursor       string `json:"end_cursor"`
}

// convertLessonEdges converts pb LessonConnection Edges to JSON format
func convertLessonEdges(edges []*pb.LessonsConnection_Edge) []LessonEdgeJSON {
	result := make([]LessonEdgeJSON, len(edges))
	for i, edge := range edges {
		result[i] = LessonEdgeJSON{
			Node:   toLessonJSON(edge.Node),
			Cursor: edge.Cursor,
		}
	}
	return result
}

// convertLessonWithProgressEdges converts pb LessonWithProgressConnection Edges to JSON format
func convertLessonWithProgressEdges(edges []*pb.LessonsWithProgressConnection_Edge) []LessonWithProgressEdgeJSON {
	result := make([]LessonWithProgressEdgeJSON, len(edges))
	for i, edge := range edges {
		baseLesson := toLessonJSON(edge.Node.Lesson)
		result[i] = LessonWithProgressEdgeJSON{
			Node: LessonWithProgressJSON{
				LessonJSON:  baseLesson,
				IsCompleted: edge.Node.IsCompleted,
				IsCurrent:   edge.Node.IsCurrent,
			},
			Cursor: edge.Cursor,
		}
	}
	return result
}

// convertCourseWithProgressEdges converts pb CoursesWithProgressConnection Edges to JSON format
func convertCourseWithProgressEdges(edges []*pb.CoursesWithProgressConnection_Edge) []CourseWithProgressEdgeJSON {
	result := make([]CourseWithProgressEdgeJSON, len(edges))
	for i, edge := range edges {
		baseCourse := toCourseJSON(edge.Node.Course)
		result[i] = CourseWithProgressEdgeJSON{
			Node: CourseWithProgressJSON{
				CourseJSON:      baseCourse,
				CurrentLessonID: edge.Node.CurrentLessonId,
			},
			Cursor: edge.Cursor,
		}
	}
	return result
}

// convertProtoTimestamp converts a protobuf Timestamp to RFC3339 string
func convertProtoTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

func toPageInfoJSON(pi *pb.PageInfo) PageInfoJSON {
	if pi == nil {
		return PageInfoJSON{}
	}

	return PageInfoJSON{
		HasPreviousPage: pi.HasPreviousPage,
		HasNextPage:     pi.HasNextPage,
		StartCursor:     pi.StartCursor,
		EndCursor:       pi.EndCursor,
	}
}

// protoToJSON converts a protobuf Timestamp pointer to RFC3339 string pointer
func protoToJSON(ts *timestamppb.Timestamp) *string {
	if ts == nil {
		return nil
	}
	s := ts.AsTime().Format(time.RFC3339)
	return &s
}

// toCourseJSON converts pb.Course to CourseJSON with proper timestamp formatting
func toCourseJSON(course *pb.Course) CourseJSON {
	return CourseJSON{
		ID:          course.Id,
		Order:       course.Order,
		Title:       course.Title,
		Description: course.Description,
		CreatedAt:   convertProtoTimestamp(course.CreatedAt),
		EditedAt:    protoToJSON(course.EditedAt),
		DeletedAt:   protoToJSON(course.DeletedAt),
	}
}

// toLessonJSON converts pb.Lesson to LessonJSON with proper timestamp formatting
func toLessonJSON(lesson *pb.Lesson) LessonJSON {
	return LessonJSON{
		ID:          lesson.Id,
		CourseID:    lesson.CourseId,
		Order:       lesson.Order,
		Title:       lesson.Title,
		Description: lesson.Description,
		Body:        lesson.Body,
		CreatedAt:   convertProtoTimestamp(lesson.CreatedAt),
		EditedAt:    protoToJSON(lesson.EditedAt),
		DeletedAt:   protoToJSON(lesson.DeletedAt),
	}
}

// TODO: Handle these errors properly
func isHandledError(ctx *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "not found") {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return true
	}

	if strings.Contains(lower, "uuid") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return true
	}

	if strings.Contains(lower, "does not exist") {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return true
	}

	return false
}

const DefaultPageSize = 20

// parsePagination extracts pagination params and validates numeric values.
// Returns a pb.Pagination or an error if parsing fails.
func parsePagination(ctx *gin.Context) (*pb.Pagination, error) {
	after := ctx.Query("after")
	before := ctx.Query("before")
	firstParam := ctx.Query("first")
	lastParam := ctx.Query("last")

	var first, last int64
	if firstParam != "" {
		v, err := strconv.ParseInt(firstParam, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		first = v
	}
	if lastParam != "" {
		v, err := strconv.ParseInt(lastParam, 10, 64)
		if err != nil {
			return nil, errors.Trace(err)
		}
		last = v
	}

	if first == 0 && last == 0 {
		first = DefaultPageSize
	}

	return &pb.Pagination{
		After:  after,
		Before: before,
		First:  first,
		Last:   last,
	}, nil
}
