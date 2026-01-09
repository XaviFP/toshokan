package graph

import (
	v1 "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/grapher/graph/model"
)

func cardsFromInput(input []*model.CreateCardInput) []*v1.Card {
	var cards []*v1.Card

	for _, card := range input {
		var answers []*v1.Answer

		for _, answer := range card.Answers {
			answers = append(answers, &v1.Answer{
				Text:      answer.Text,
				IsCorrect: answer.IsCorrect,
			})
		}

		var explanation string
		if card.Explanation != nil {
			explanation = *card.Explanation
		}

		cards = append(cards, &v1.Card{
			Title:           card.Title,
			PossibleAnswers: answers,
			Explanation:     explanation,
		})
	}

	return cards
}

func cardToModel(in *v1.Card) *model.Card {
	var answers []*model.Answer

	for _, a := range in.PossibleAnswers {
		answers = append(answers, &model.Answer{
			ID:        a.Id,
			Text:      a.Text,
			IsCorrect: a.IsCorrect,
		})
	}

	return &model.Card{
		ID:          in.Id,
		Title:       in.Title,
		Answers:     answers,
		Explanation: &in.Explanation,
	}
}

func cardsToModel(in []*v1.Card) []*model.Card {
	var cards []*model.Card

	for _, c := range in {
		var outAnswers []*model.Answer
		for _, a := range c.PossibleAnswers {
			outAnswers = append(outAnswers, &model.Answer{
				ID:        a.Id,
				Text:      a.Text,
				IsCorrect: a.IsCorrect,
			})
		}

		cards = append(cards, &model.Card{
			ID:          c.Id,
			Title:       c.Title,
			Answers:     outAnswers,
			Explanation: &c.Explanation,
			Kind:        c.Kind,
		})
	}

	return cards
}

func deckToModel(in *v1.Deck) *model.Deck {
	return &model.Deck{
		ID:          in.Id,
		Title:       in.Title,
		Description: in.Description,
		Cards:       cardsToModel(in.Cards),
	}
}
