package grpc

import (
	"context"
	"net"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/deck/internal/deck"
)

type Server struct {
	GRPCAddr      string
	GRPCTransport string

	Repository deck.Repository

	grpcServer *grpc.Server
}

func (s *Server) Start() error {
	s.grpcServer = grpc.NewServer()
	pb.RegisterDecksAPIServer(s.grpcServer, s)

	listener, err := net.Listen(s.GRPCTransport, s.GRPCAddr)
	if err != nil {
		return errors.Annotatef(err, "failed to listen on %s", listener.Addr())
	}
	defer listener.Close()

	if err := s.grpcServer.Serve(listener); err != nil {
		return errors.Annotatef(err, "failed to serve on %s", listener.Addr())
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) GetDeck(ctx context.Context, req *pb.GetDeckRequest) (*pb.GetDeckResponse, error) {
	deckID, err := uuid.Parse(req.DeckId)
	if err != nil {
		return &pb.GetDeckResponse{}, errors.Trace(err)
	}

	d, err := s.Repository.GetDeck(ctx, deckID)
	if err != nil {
		if errors.Cause(err) == deck.ErrDeckNotFound {

			return &pb.GetDeckResponse{}, status.Error(codes.NotFound, errors.Trace(err).Error())
		}
		return &pb.GetDeckResponse{}, errors.Trace(err)

	}

	return &pb.GetDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) GetDecks(ctx context.Context, req *pb.GetDecksRequest) (*pb.GetDecksResponse, error) {
	decks, err := s.Repository.GetDecks(ctx)
	if err != nil {
		return &pb.GetDecksResponse{}, errors.Trace(err)
	}

	return &pb.GetDecksResponse{Decks: toGRPCDecks(decks)}, nil
}

func (s *Server) CreateDeck(ctx context.Context, req *pb.CreateDeckRequest) (*pb.CreateDeckResponse, error) {
	d, err := fromGRPCDeck(req.Deck)
	if err != nil {
		return nil, errors.Trace(err)
	}

	isValid, _ := d.Validate()
	if !isValid {
		return &pb.CreateDeckResponse{}, deck.ErrDeckInvalid
	}

	d.GenerateUUIDs()

	if err := s.Repository.StoreDeck(ctx, d); err != nil {
		return &pb.CreateDeckResponse{}, errors.Trace(err)
	}

	return &pb.CreateDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) DeleteDeck(ctx context.Context, req *pb.DeleteDeckRequest) (*pb.DeleteDeckResponse, error) {
	deckID, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.DeleteDeckResponse{}, errors.Trace(err)
	}

	err = s.Repository.DeleteDeck(ctx, deckID)

	return &pb.DeleteDeckResponse{}, errors.Trace(err)
}

func toGRPCDecks(decks []deck.Deck) []*pb.Deck {
	var out = make([]*pb.Deck, 0, len(decks))

	for _, deck := range decks {
		out = append(out, toGRPCDeck(deck))
	}

	return out
}

func toGRPCDeck(d deck.Deck) *pb.Deck {
	return &pb.Deck{
		Id:          d.ID.String(),
		AuthorId:    d.AuthorID.String(),
		Title:       d.Title,
		Description: d.Description,
		Cards:       toGRPCCards(d.Cards),
	}
}

func toGRPCCards(cards []deck.Card) []*pb.Card {
	var out = make([]*pb.Card, 0, len(cards))

	for _, c := range cards {
		out = append(out, &pb.Card{
			Id:              c.ID.String(),
			Title:           c.Title,
			PossibleAnswers: toGRPCAnswers(c.PossibleAnswers),
		})
	}

	return out
}

func toGRPCAnswers(answers []deck.Answer) []*pb.Answer {
	var out = make([]*pb.Answer, 0, len(answers))

	for _, a := range answers {
		out = append(out, &pb.Answer{
			Id:        a.ID.String(),
			Text:      a.Text,
			IsCorrect: a.IsCorrect,
		})
	}

	return out
}

func fromGRPCDeck(d *pb.Deck) (deck.Deck, error) {
	if d == nil {
		return deck.Deck{}, nil
	}

	var deckID uuid.UUID

	if d.Id != "" {
		var err error

		deckID, err = uuid.Parse(d.Id)
		if err != nil {
			return deck.Deck{}, errors.Trace(err)
		}

	}

	authorID, err := uuid.Parse(d.AuthorId)
	if err != nil {
		return deck.Deck{}, errors.Trace(err)
	}

	cards, err := fromGRPCCards(d.Cards)
	if err != nil {
		return deck.Deck{}, errors.Trace(err)
	}

	return deck.Deck{
		ID:          deckID,
		AuthorID:    authorID,
		Title:       d.Title,
		Description: d.Description,
		Cards:       cards,
	}, nil
}

func fromGRPCCards(cards []*pb.Card) ([]deck.Card, error) {
	var out = make([]deck.Card, 0, len(cards))

	for _, c := range cards {
		converted, err := fromGRPCCard(c)
		if err != nil {
			return []deck.Card{}, errors.Trace(err)
		}

		out = append(out, converted)
	}

	return out, nil
}

func fromGRPCCard(c *pb.Card) (deck.Card, error) {
	var cardID uuid.UUID

	if c.Id != "" {
		var err error

		cardID, err = uuid.Parse(c.Id)
		if err != nil {
			return deck.Card{}, errors.Trace(err)
		}
	}

	answers, err := fromGRPCAnswers(c.PossibleAnswers)
	if err != nil {
		return deck.Card{}, errors.Trace(err)
	}

	return deck.Card{
		ID:              cardID,
		Title:           c.Title,
		PossibleAnswers: answers,
	}, nil
}

func fromGRPCAnswers(answers []*pb.Answer) ([]deck.Answer, error) {
	var out = make([]deck.Answer, 0, len(answers))

	for _, a := range answers {
		var answerID uuid.UUID
		if a.Id != "" {
			var err error

			answerID, err = uuid.Parse(a.Id)
			if err != nil {
				return []deck.Answer{}, errors.Trace(err)
			}
		}

		out = append(out, deck.Answer{
			ID:        answerID,
			Text:      a.Text,
			IsCorrect: a.IsCorrect,
		})
	}

	return out, nil
}
