package postgres

import (
	"database/sql"
	"fmt"
	"github.com/rtrzebinski/simple-memorizer-4/internal/models"
)

type Reader struct {
	db *sql.DB
}

func NewReader(db *sql.DB) *Reader {
	return &Reader{db: db}
}

func (r *Reader) AllExercises() (models.Exercises, error) {
	var exercises models.Exercises

	const query = `
		SELECT e.id, e.question, e.answer, COALESCE(er.bad_answers, 0), COALESCE(er.good_answers, 0) 
		FROM exercise e
		LEFT JOIN exercise_result er on e.id = er.exercise_id
		ORDER BY e.id DESC
		`

	rows, err := r.db.Query(query)
	if err != nil {
		return exercises, err
	}

	for rows.Next() {
		var exercise models.Exercise

		err = rows.Scan(&exercise.Id, &exercise.Question, &exercise.Answer, &exercise.BadAnswers, &exercise.GoodAnswers)
		if err != nil {
			return exercises, err
		}

		exercises = append(exercises, exercise)
	}

	return exercises, nil
}

func (r *Reader) RandomExercise() (models.Exercise, error) {
	var exercise models.Exercise

	const query = `
		SELECT e.id, e.question, e.answer, COALESCE(er.bad_answers, 0), COALESCE(er.good_answers, 0) 
		FROM exercise e
		LEFT JOIN exercise_result er on e.id = er.exercise_id
		ORDER BY random()
		LIMIT 1`

	if err := r.db.QueryRow(query).Scan(&exercise.Id, &exercise.Question, &exercise.Answer, &exercise.BadAnswers, &exercise.GoodAnswers); err != nil {
		return exercise, fmt.Errorf("failed to scan query results: %w", err)
	}

	return exercise, nil
}
