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
	ID   uint
	Name string
}

type Book struct {
	ID       uint
	AuthorID *uint
	Title    string
}

type Chapter struct {
	ID     uint
	BookID uint
	Title  string
}

type TurboChapter turbo.Turbo[*Chapter]

var NewTurboChapters = turbo.NewConstructor(func(tb *turbo.Turbo[*Chapter]) *TurboChapter { return (*TurboChapter)(tb) })

type TurboBook turbo.Turbo[*Book]

var NewTurboBooks = turbo.NewConstructor(func(tb *turbo.Turbo[*Book]) *TurboBook { return (*TurboBook)(tb) })

type TurboAuthor turbo.Turbo[*Author]

var NewTurboAuthors = turbo.NewConstructor(func(tb *turbo.Turbo[*Author]) *TurboAuthor { return (*TurboAuthor)(tb) })

func (t *TurboAuthor) Books(ctx context.Context, db *gorm.DB) ([]*TurboBook, error) {
	return turbo.LoadRelation(ctx, t.Loader, "books", t.Model, func(ctx context.Context, authors []*Author) (turbo.RelationLookupFunc[*Author, []*TurboBook], error) {
		var authorIDs []uint
		for _, author := range authors {
			authorIDs = append(authorIDs, author.ID)
		}
		var books []*Book
		if err := db.Debug().Where("author_id IN (?)", authorIDs).Find(&books).Error; err != nil {
			return nil, err
		}
		turboBooks := NewTurboBooks(books)
		var booksByAuthorID = make(map[uint][]*TurboBook)
		for _, book := range turboBooks {
			tb := (*TurboBook)(book)
			booksByAuthorID[*book.Model.AuthorID] = append(booksByAuthorID[*book.Model.AuthorID], tb)
		}
		return func(author *Author) []*TurboBook {
			return booksByAuthorID[author.ID]
		}, nil
	})
}

func (t *TurboAuthor) BooksChildren(ctx context.Context, db *gorm.DB) ([]*TurboBook, error) {
	return turbo.LoadChildren(ctx, t.Loader, "books", t.Model, turbo.LoadChildrenArgs[*Author, *Book, *TurboBook]{
		ModelIDFunc: func(author *Author) uint { return author.ID },
		QueryChildrenFunc: func(ctx context.Context, authorIDs []uint) ([]*Book, error) {
			var books []*Book
			if err := db.Debug().Where("author_id IN (?)", authorIDs).Find(&books).Error; err != nil {
				return nil, err
			}
			return books, nil
		},
		TurboConstructor: NewTurboBooks,
		ParentIDFunc:     func(book *TurboBook) uint { return *book.Model.AuthorID },
	})
}

func (t *TurboBook) Chapters(ctx context.Context, db *gorm.DB) ([]*TurboChapter, error) {
	return turbo.LoadRelation(ctx, t.Loader, "chapters", t.Model, func(ctx context.Context, books []*Book) (turbo.RelationLookupFunc[*Book, []*TurboChapter], error) {
		var bookIDs []uint
		for _, book := range books {
			bookIDs = append(bookIDs, book.ID)
		}
		var chapters []*Chapter
		if err := db.Debug().Where("book_id IN (?)", bookIDs).Find(&chapters).Error; err != nil {
			return nil, err
		}
		turboChapters := NewTurboChapters(chapters)
		var chaptersByBookID = make(map[uint][]*TurboChapter)
		for _, chapter := range turboChapters {
			tb := (*TurboChapter)(chapter)
			chaptersByBookID[chapter.Model.BookID] = append(chaptersByBookID[chapter.Model.BookID], tb)
		}
		return func(book *Book) []*TurboChapter {
			return chaptersByBookID[book.ID]
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
		fmt.Println(turboAuthor.Model.Name)
		books, err := turboAuthor.BooksChildren(ctx, db)
		if err != nil {
			t.Fatalf("Failed to get books: %v", err)
		}
		for _, book := range books {
			fmt.Println("  ", book.Model.Title)
			chapters, err := book.Chapters(ctx, db)
			if err != nil {
				t.Fatalf("Failed to get chapters: %v", err)
			}
			for _, chapter := range chapters {
				fmt.Println("    ", chapter.Model.Title)
			}
		}
	}
}

func printJSON(v any) {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(json))
}
