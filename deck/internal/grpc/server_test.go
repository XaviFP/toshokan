package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/XaviFP/toshokan/common/pagination"
	pb "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/deck/internal/deck"
)

func TestServer_GetDeck(t *testing.T) {
	repoMock := &deck.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		deckID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		deck := deck.Deck{
			ID:          deckID,
			Title:       "Go Learning",
			Description: "Polish your Go skills",
			AuthorID:    uuid.MustParse("f3b59a97-e678-4410-8ed2-f1094a234a01"),
			Public:      true,
			Cards: []deck.Card{
				{
					Title: "What does CSP stand for?",
					PossibleAnswers: []deck.Answer{
						{Text: "Communicating Sequential Processes", IsCorrect: true},
					},
					Kind: "single_choice",
				},
			}}
		repoMock.On("GetDeck", mock.Anything, deckID).Return(deck, nil)
		req := pb.GetDeckRequest{
			DeckId: deckID.String(),
			UserId: uuid.MustParse("32571d06-54fb-4d5a-8c6b-dfdc8a51e1a1").String(),
		}

		res, err := srv.GetDeck(context.Background(), &req)
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetDeckResponse{Deck: toGRPCDeck(deck)}, res)
	})

	t.Run("failure", func(t *testing.T) {
		deckID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		repoMock.On("GetDeck", mock.Anything, deckID).Return(deck.Deck{}, assert.AnError)
		req := pb.GetDeckRequest{DeckId: deckID.String()}

		res, err := srv.GetDeck(context.Background(), &req)
		assert.Error(t, err)
		assert.Equal(t, &pb.GetDeckResponse{}, res)
	})
}

func TestServer_GetDecks(t *testing.T) {
	repoMock := &deck.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		d := deck.Deck{
			ID:          uuid.MustParse("5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598"),
			Title:       "Go Learning",
			Description: "Polish your Go skills",
			AuthorID:    uuid.MustParse("f3b59a97-e678-4410-8ed2-f1094a234a01"),
			Cards: []deck.Card{
				{
					Title: "What does CSP stand for?",
					PossibleAnswers: []deck.Answer{
						{Text: "Communicating Sequential Processes", IsCorrect: true},
					},
					Kind: "single_choice",
				},
				{
					Title: "Which is the underlying data type of a slice in Go?",
					PossibleAnswers: []deck.Answer{
						{Text: "Map"},
						{Text: "Linked list"},
						{Text: "Array", IsCorrect: true},
					},
					Kind: "single_choice",
				},
			},
		}

		decks := map[uuid.UUID]deck.Deck{
			d.ID: d,
		}

		repoMock.On("GetDecks", mock.Anything, []uuid.UUID{d.ID}).Return(decks, nil)

		res, err := srv.GetDecks(context.Background(), &pb.GetDecksRequest{DeckIds: []string{"5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598"}})
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetDecksResponse{Decks: map[string]*pb.Deck{
			"5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598": toGRPCDeck(d),
		}}, res)
	})

	t.Run("error", func(t *testing.T) {
		repoMock.On("GetDecks", mock.Anything, mock.Anything).Return(map[uuid.UUID]deck.Deck{}, assert.AnError)

		_, err := srv.GetDecks(context.Background(), &pb.GetDecksRequest{})
		assert.Error(t, err)
	})

	repoMock.AssertExpectations(t)
}

func TestServer_CreateDeck(t *testing.T) {
	successTitle := "Go Learning success"
	failureTitle := "Go Learning failure"
	d := deck.Deck{Description: "Polish your Go skills", Cards: []deck.Card{
		{
			Title: "What does CSP stand for?",
			PossibleAnswers: []deck.Answer{
				{Text: "Communicating Sequential Processes", IsCorrect: true},
			},
			Kind: "single_choice",
		},
	}}

	t.Run("success", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		repoMock.On("StoreDeck", mock.Anything, mock.MatchedBy(func(d deck.Deck) bool { return d.Title == successTitle })).Return(nil)
		srv := &Server{Repository: repoMock}

		d.Title = successTitle
		deckReq := pb.CreateDeckRequest{Deck: toGRPCDeck(d)}

		actualRes, err := srv.CreateDeck(context.Background(), &deckReq)
		assert.NoError(t, err)
		expectedRes := &pb.CreateDeckResponse{Deck: toGRPCDeck(d)}
		assert.Equal(t, expectedRes.Deck.Title, actualRes.Deck.Title)
		assert.Equal(t, expectedRes.Deck.Description, actualRes.Deck.Description)
		assert.Equal(t, expectedRes.Deck.Cards[0].Title, actualRes.Deck.Cards[0].Title)
	})

	t.Run("failure", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		repoMock.On("StoreDeck", mock.Anything, mock.MatchedBy(func(d deck.Deck) bool { return d.Title == failureTitle })).Return(assert.AnError)

		srv := &Server{Repository: repoMock}
		d.Title = failureTitle
		deckReq := pb.CreateDeckRequest{Deck: toGRPCDeck(d)}

		res, err := srv.CreateDeck(context.Background(), &deckReq)
		assert.Error(t, err)
		assert.Equal(t, &pb.CreateDeckResponse{}, res)
	})
}

func TestServer_DeleteDeck(t *testing.T) {
	repoMock := &deck.RepositoryMock{}
	srv := &Server{Repository: repoMock}
	validDeckID := "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"
	invalidDeckID := "1f30a72f-5d7a-48da-a5c2-42efece6972a"

	t.Run("success", func(t *testing.T) {
		repoMock.On("DeleteDeck", mock.Anything, uuid.MustParse(validDeckID)).Return(nil)
		deckReq := pb.DeleteDeckRequest{Id: validDeckID}
		res, err := srv.DeleteDeck(context.Background(), &deckReq)
		assert.NoError(t, err)
		assert.Equal(t, &pb.DeleteDeckResponse{}, res)
	})

	t.Run("failure", func(t *testing.T) {
		repoMock.On("DeleteDeck", mock.Anything, uuid.MustParse(invalidDeckID)).Return(assert.AnError)
		deckReq := pb.DeleteDeckRequest{Id: invalidDeckID}
		res, err := srv.DeleteDeck(context.Background(), &deckReq)
		assert.Error(t, err)
		assert.Equal(t, &pb.DeleteDeckResponse{}, res)
	})
}

func TestServer_GetPopularDecks(t *testing.T) {
	repoMock := &deck.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	userID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		repoMock.On("GetPopularDecks", mock.Anything, userID, pagination.Pagination{
			First: 2,
			After: pagination.Cursor("9999"),
		},
		).Return(deck.PopularDecksConnection{
			Edges: []deck.PopularDeckEdge{
				{
					DeckID: uuid.MustParse("f79aea77-9aa0-4a84-b4c8-d000a27d2c52"),
					Cursor: pagination.Cursor("9999"),
				},
				{
					DeckID: uuid.MustParse("6363e2c6-d89e-4610-92e8-1e1d2fea49ec"),
					Cursor: pagination.Cursor("8888"),
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage: true,
				StartCursor: pagination.Cursor("9999"),
				EndCursor:   pagination.Cursor("8888"),
			},
		}, nil)

		res, err := srv.GetPopularDecks(context.Background(), &pb.GetPopularDecksRequest{
			UserId: userID.String(),
			Pagination: &pb.Pagination{
				First: 2,
				After: "9999",
			},
		})
		assert.NoError(t, err)

		assert.Equal(t, &pb.GetPopularDecksResponse{Connection: &pb.PopularDecksConnection{
			Edges: []*pb.PopularDecksConnection_Edge{
				{
					DeckId: "f79aea77-9aa0-4a84-b4c8-d000a27d2c52",
					Cursor: "9999",
				},
				{
					DeckId: "6363e2c6-d89e-4610-92e8-1e1d2fea49ec",
					Cursor: "8888",
				},
			},
			PageInfo: &pb.PageInfo{
				HasNextPage: true,
				StartCursor: "9999",
				EndCursor:   "8888",
			},
		}}, res)
	})

	t.Run("error", func(t *testing.T) {
		repoMock.On("GetPopularDecks", mock.Anything, userID, pagination.Pagination{}).Return(
			deck.PopularDecksConnection{},
			assert.AnError,
		)
		_, err := srv.GetPopularDecks(context.Background(), &pb.GetPopularDecksRequest{UserId: userID.String(), Pagination: &pb.Pagination{}})
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestServer_GetCards(t *testing.T) {
	repoMock := &deck.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		c := deck.Card{
			ID:    uuid.MustParse("5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598"),
			Title: "What does CSP stand for?",
			PossibleAnswers: []deck.Answer{
				{Text: "Communicating Sequential Processes", IsCorrect: true},
			},
			Kind: "single_choice",
		}

		cards := map[uuid.UUID]deck.Card{
			c.ID: c,
		}

		repoMock.On("GetCards", mock.Anything, []uuid.UUID{c.ID}).Return(cards, nil)

		res, err := srv.GetCards(context.Background(), &pb.GetCardsRequest{CardIds: []string{"5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598"}})
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetCardsResponse{Cards: map[string]*pb.Card{
			"5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598": {
				Id:              c.ID.String(),
				Title:           c.Title,
				Explanation:     c.Explanation,
				PossibleAnswers: toGRPCAnswers(c.PossibleAnswers),
				Kind:            "single_choice",
			},
		}}, res)
	})

	t.Run("error", func(t *testing.T) {
		repoMock.On("GetCards", mock.Anything, mock.Anything).Return(map[uuid.UUID]deck.Card{}, assert.AnError)

		_, err := srv.GetCards(context.Background(), &pb.GetCardsRequest{})
		assert.Error(t, err)
	})

	repoMock.AssertExpectations(t)
}

func TestServer_UpdateDeck(t *testing.T) {
	validDeckID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success_update_title", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Title"
		repoMock.On("UpdateDeck", mock.Anything, validDeckID, deck.DeckUpdates{
			Title: &newTitle,
		}).Return(deck.Deck{
			ID:          validDeckID,
			Title:       newTitle,
			Description: "Original description",
		}, nil)

		res, err := srv.UpdateDeck(context.Background(), &pb.UpdateDeckRequest{
			Id:    validDeckID.String(),
			Title: &newTitle,
		})

		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Deck.Title)
		assert.Equal(t, "Original description", res.Deck.Description)
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		_, err := srv.UpdateDeck(context.Background(), &pb.UpdateDeckRequest{
			Id: validDeckID.String(),
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no fields to update")
	})

	t.Run("failure_deck_not_found", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Title"
		repoMock.On("UpdateDeck", mock.Anything, validDeckID, mock.Anything).Return(deck.Deck{}, deck.ErrDeckNotFound)

		_, err := srv.UpdateDeck(context.Background(), &pb.UpdateDeckRequest{
			Id:    validDeckID.String(),
			Title: &newTitle,
		})

		assert.Error(t, err)
	})

	t.Run("failure_invalid_deck_ID", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Title"
		_, err := srv.UpdateDeck(context.Background(), &pb.UpdateDeckRequest{
			Id:    "invalid-uuid",
			Title: &newTitle,
		})

		assert.Error(t, err)
	})
}

func TestServer_UpdateCard(t *testing.T) {
	validDeckID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	validCardID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")

	t.Run("success_update_title", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Card Title"
		repoMock.On("UpdateCard", mock.Anything, validDeckID, validCardID, deck.CardUpdates{
			Title: &newTitle,
		}).Return(deck.Card{
			ID:    validCardID,
			Title: newTitle,
			Kind:  "single_choice",
		}, nil)

		res, err := srv.UpdateCard(context.Background(), &pb.UpdateCardRequest{
			DeckId: validDeckID.String(),
			CardId: validCardID.String(),
			Title:  &newTitle,
		})

		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Card.Title)
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		_, err := srv.UpdateCard(context.Background(), &pb.UpdateCardRequest{
			DeckId: validDeckID.String(),
			CardId: validCardID.String(),
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no fields to update")
	})

	t.Run("failure_invalid_kind", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		invalidKind := "invalid_kind"
		_, err := srv.UpdateCard(context.Background(), &pb.UpdateCardRequest{
			DeckId: validDeckID.String(),
			CardId: validCardID.String(),
			Kind:   &invalidKind,
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, deck.ErrInvalidKind)
	})

	t.Run("failure_card_not_found", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Card Title"
		repoMock.On("UpdateCard", mock.Anything, validDeckID, validCardID, mock.Anything).Return(deck.Card{}, deck.ErrCardNotFound)

		_, err := srv.UpdateCard(context.Background(), &pb.UpdateCardRequest{
			DeckId: validDeckID.String(),
			CardId: validCardID.String(),
			Title:  &newTitle,
		})

		assert.Error(t, err)
	})
}

func TestServer_UpdateAnswer(t *testing.T) {
	validDeckID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	validCardID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
	validAnswerID := uuid.MustParse("5ec790fb-3dcc-4ee4-8c6d-daa9e4e11598")

	t.Run("success_update_text", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newText := "Updated Answer Text"
		repoMock.On("UpdateAnswer", mock.Anything, validDeckID, validCardID, validAnswerID, deck.AnswerUpdates{
			Text: &newText,
		}).Return(deck.Answer{
			ID:        validAnswerID,
			Text:      newText,
			IsCorrect: true,
		}, nil)

		res, err := srv.UpdateAnswer(context.Background(), &pb.UpdateAnswerRequest{
			DeckId:   validDeckID.String(),
			CardId:   validCardID.String(),
			AnswerId: validAnswerID.String(),
			Text:     &newText,
		})

		assert.NoError(t, err)
		assert.Equal(t, newText, res.Answer.Text)
		assert.True(t, res.Answer.IsCorrect)
	})

	t.Run("success_update_is_correct", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		isCorrect := false
		repoMock.On("UpdateAnswer", mock.Anything, validDeckID, validCardID, validAnswerID, deck.AnswerUpdates{
			IsCorrect: &isCorrect,
		}).Return(deck.Answer{
			ID:        validAnswerID,
			Text:      "Original text",
			IsCorrect: isCorrect,
		}, nil)

		res, err := srv.UpdateAnswer(context.Background(), &pb.UpdateAnswerRequest{
			DeckId:    validDeckID.String(),
			CardId:    validCardID.String(),
			AnswerId:  validAnswerID.String(),
			IsCorrect: &isCorrect,
		})

		assert.NoError(t, err)
		assert.Equal(t, "Original text", res.Answer.Text)
		assert.False(t, res.Answer.IsCorrect)
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		_, err := srv.UpdateAnswer(context.Background(), &pb.UpdateAnswerRequest{
			DeckId:   validDeckID.String(),
			CardId:   validCardID.String(),
			AnswerId: validAnswerID.String(),
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no fields to update")
	})

	t.Run("failure_invalid_answer_ID", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newText := "Updated Answer Text"
		_, err := srv.UpdateAnswer(context.Background(), &pb.UpdateAnswerRequest{
			DeckId:   validDeckID.String(),
			CardId:   validCardID.String(),
			AnswerId: "invalid-uuid",
			Text:     &newText,
		})

		assert.Error(t, err)
	})
}
