package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rtrzebinski/simple-memorizer-4/internal/backend/storage/entities"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"time"
)

func findExerciseById(db *sql.DB, exerciseId int) *entities.Exercise {

	const query = `
		SELECT e.id, e.question, e.answer
		FROM exercise e
		WHERE e.id = $1;`

	rows, err := db.Query(query, exerciseId)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var exercise entities.Exercise

		err = rows.Scan(&exercise.Id, &exercise.Question, &exercise.Answer)
		if err != nil {
			panic(err)
		}

		return &exercise
	}

	return nil
}

func fetchLatestExercise(db *sql.DB) entities.Exercise {
	var exercise entities.Exercise

	const query = `
		SELECT e.id, e.question, e.answer
		FROM exercise e
		ORDER BY id DESC
		LIMIT 1;`

	if err := db.QueryRow(query).Scan(&exercise.Id, &exercise.Question, &exercise.Answer); err != nil {
		panic(err)
	}

	return exercise
}

func storeExercise(db *sql.DB, exercise *entities.Exercise) {
	query := `INSERT INTO exercise (question, answer) VALUES ($1, $2) RETURNING id;`

	err := db.QueryRow(query, &exercise.Question, &exercise.Answer).Scan(&exercise.Id)
	if err != nil {
		panic(err)
	}
}

func findExerciseResultByExerciseId(db *sql.DB, exerciseId int) entities.ExerciseResult {
	var exerciseResult entities.ExerciseResult

	const query = `
		SELECT er.id, er.exercise_id, er.bad_answers, er.good_answers
		FROM exercise_result er
		WHERE er.exercise_id = $1;`

	if err := db.QueryRow(query, exerciseId).Scan(&exerciseResult.Id, &exerciseResult.ExerciseId, &exerciseResult.BadAnswers, &exerciseResult.GoodAnswers); err != nil {
		panic(err)
	}

	return exerciseResult
}

func createPostgresContainer(ctx context.Context, dbname string) (testcontainers.Container, *sql.DB, error) {
	env := map[string]string{
		"POSTGRES_PASSWORD": "password",
		"POSTGRES_USER":     "postgres",
		"POSTGRES_DB":       dbname,
	}

	port := "5432/tcp"
	dbURL := func(host string, port nat.Port) string {
		return fmt.Sprintf("postgres://postgres:password@%s:%s/%s?sslmode=disable", host, port.Port(), dbname)
	}

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{port},
			Cmd:          []string{"postgres", "-c", "fsync=off"},
			Env:          env,
			SkipReaper:   true,
			WaitingFor: wait.ForAll(
				wait.ForSQL(nat.Port(port), "postgres", dbURL).WithStartupTimeout(time.Second*5),
				wait.ForLog("database system is ready to accept connections"),
			),
		},
		Started: true,
	}

	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return container, nil, fmt.Errorf("failed to start container: %s", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return container, nil, fmt.Errorf("failed to get container external port: %s", err)
	}

	log.Println("postgres container ready and running at port: ", mappedPort)

	url := fmt.Sprintf("postgres://postgres:password@localhost:%s/%s?sslmode=disable", mappedPort.Port(), dbname)

	db, err := sql.Open("postgres", url)
	if err != nil {
		return container, db, fmt.Errorf("failed to establish database connection: %s", err)
	}

	return container, db, nil
}

func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := migrate_postgres.WithInstance(db, &migrate_postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create migrator driver: %s", err)
	}

	return migrate.NewWithDatabaseInstance("file://../../../../migrations", "postgres", driver)
}
