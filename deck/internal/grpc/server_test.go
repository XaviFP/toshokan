package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

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
	t.Run("success", func(t *testing.T) {
		decks := []deck.Deck{
			{
				Title:       "Go Learning",
				Description: "Polish your Go skills",
				AuthorID:    uuid.MustParse("f3b59a97-e678-4410-8ed2-f1094a234a01"),
				Cards: []deck.Card{
					{
						Title: "What does CSP stand for?",
						PossibleAnswers: []deck.Answer{
							{Text: "Communicating Sequential Processes", IsCorrect: true},
						}},
					{
						Title: "Which is the underlying data type of a slice in Go?",
						PossibleAnswers: []deck.Answer{
							{Text: "Map", IsCorrect: false},
							{Text: "Linked list", IsCorrect: false},
							{Text: "Array", IsCorrect: true},
						}},
				}},
			{
				Title:       "Go Learning",
				Description: "Polish your Go skills",
				AuthorID:    uuid.MustParse("f3b59a97-e678-4410-8ed2-f1094a234a01"),
				Cards: []deck.Card{
					{
						Title: "What does CSP stand for?",
						PossibleAnswers: []deck.Answer{
							{Text: "Communicating Sequential Processes", IsCorrect: true},
						}},
					{
						Title: "Which is the underlying data type of a slice in Go?",
						PossibleAnswers: []deck.Answer{
							{Text: "Map", IsCorrect: false},
							{Text: "Linked list", IsCorrect: false},
							{Text: "Array", IsCorrect: true},
						}},
				}},
		}

		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("GetDecks", mock.Anything).Return(decks, nil)

		res, err := srv.GetDecks(context.Background(), &pb.GetDecksRequest{UserId: "f3b59a97-e678-4410-8ed2-f1094a234a01"})
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetDecksResponse{Decks: toGRPCDecks(decks)}, res)
	})

	t.Run("failure", func(t *testing.T) {
		repoMock := &deck.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("GetDecks", mock.Anything).Return([]deck.Deck{}, assert.AnError)

		res, err := srv.GetDecks(context.Background(), &pb.GetDecksRequest{})
		assert.Error(t, err)
		assert.Equal(t, &pb.GetDecksResponse{}, res)
	})
}

func TestServer_CreateDeck(t *testing.T) {
	successTitle := "Go Learning success"
	failureTitle := "Go Learning failure"
	d := deck.Deck{Description: "Polish your Go skills", Cards: []deck.Card{
		{
			Title: "What does CSP stand for?",
			PossibleAnswers: []deck.Answer{
				{Text: "Communicating Sequential Processes", IsCorrect: true},
			}},
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
