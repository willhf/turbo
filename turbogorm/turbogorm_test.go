package turbogorm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/willhf/turbo"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Author struct {
	ID   int
	Name string
}

type Book struct {
	ID       int
	AuthorID int
	Title    string
}

type TurboBook turbo.Turbo[*Book]

var NewTurboBooks = turbo.NewConstructor(func(tb *turbo.Turbo[*Book]) *TurboBook { return (*TurboBook)(tb) })

type TurboAuthor turbo.Turbo[*Author]

var NewTurboAuthors = turbo.NewConstructor(func(tb *turbo.Turbo[*Author]) *TurboAuthor { return (*TurboAuthor)(tb) })

func (t *TurboAuthor) Books(ctx context.Context, db *gorm.DB) ([]*TurboBook, error) {
	return turbo.LoadRelation(ctx, t.Loader, "books", t.Model, func(ctx context.Context, authors []*Author) (turbo.RelationLookupFunc[*Author, []*TurboBook], error) {
		var authorIDs []int
		for _, author := range authors {
			authorIDs = append(authorIDs, author.ID)
		}
		var books []*Book
		if err := db.Debug().Where("author_id IN (?)", authorIDs).Find(&books).Error; err != nil {
			return nil, err
		}
		turboBooks := NewTurboBooks(books)
		var booksByAuthorID = make(map[int][]*TurboBook)
		for _, book := range turboBooks {
			tb := (*TurboBook)(book)
			booksByAuthorID[book.Model.AuthorID] = append(booksByAuthorID[book.Model.AuthorID], tb)
		}
		return func(author *Author) []*TurboBook {
			return booksByAuthorID[author.ID]
		}, nil
	})
}

var db *gorm.DB = nil

func TestMain(m *testing.M) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbName)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestTurbogorm(t *testing.T) {
	ctx := context.Background()

	var authors []*Author
	if err := db.Find(&authors).Error; err != nil {
		t.Fatalf("Failed to find authors: %v", err)
	}

	turboAuthors := NewTurboAuthors(authors)
	for _, turboAuthor := range turboAuthors {
		books, err := turboAuthor.Books(ctx, db)
		if err != nil {
			t.Fatalf("Failed to get books: %v", err)
		}
		fmt.Println(turboAuthor.Model.Name)
		printJSON(books)
	}
}

func printJSON(v any) {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(json))
}
