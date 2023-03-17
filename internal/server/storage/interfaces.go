package storage

import "github.com/rtrzebinski/simple-memorizer-4/internal/models"

type Reader interface {
	RandomExercise() models.Exercise
}

type Writer interface {
	IncrementBadAnswers(exerciseId int)
	IncrementGoodAnswers(exerciseId int)
}
