package turbogorm

import (
	"context"
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
	return turbo.LoadChildren(ctx, t.Loader, "books", t.Model, turbo.LoadChildrenArgs[uint, *Author, *Book, *TurboBook]{
		ModelIDFunc: func(author *Author) uint { return author.ID },
		QueryChildrenFunc: func(ctx context.Context, authorIDs []uint) ([]*Book, error) {
			var books []*Book
			if err := db.WithContext(ctx).Debug().Where("author_id IN (?)", authorIDs).Find(&books).Error; err != nil {
				return nil, err
			}
			return books, nil
		},
		TurboConstructor: NewTurboBooks,
		ParentIDFunc:     func(book *TurboBook) uint { return *book.Model.AuthorID },
	})
}

func (t *TurboBook) Chapters(ctx context.Context, db *gorm.DB) ([]*TurboChapter, error) {
	return turbo.LoadChildren(ctx, t.Loader, "chapters", t.Model, turbo.LoadChildrenArgs[uint, *Book, *Chapter, *TurboChapter]{
		ModelIDFunc: func(book *Book) uint { return book.ID },
		QueryChildrenFunc: func(ctx context.Context, bookIDs []uint) ([]*Chapter, error) {
			var chapters []*Chapter
			if err := db.WithContext(ctx).Debug().Where("book_id IN (?)", bookIDs).Find(&chapters).Error; err != nil {
				return nil, err
			}
			return chapters, nil
		},
		TurboConstructor: NewTurboChapters,
		ParentIDFunc:     func(chapter *TurboChapter) uint { return chapter.Model.BookID },
	})
}

func (t *TurboBook) Author(ctx context.Context, db *gorm.DB) (*TurboAuthor, error) {
	return turbo.LoadRelation(ctx, t.Loader, "author", t.Model, func(ctx context.Context, books []*Book) (turbo.RelationLookupFunc[*Book, *TurboAuthor], error) {
		var authorIDs []uint
		for _, book := range books {
			authorIDs = append(authorIDs, safePtr(book.AuthorID))
		}
		var authors []*Author
		if err := db.WithContext(ctx).Debug().Where("id IN (?)", authorIDs).Find(&authors).Error; err != nil {
			return nil, err
		}
		turboAuthors := NewTurboAuthors(authors)
		turboAuthorsByID := make(map[uint]*TurboAuthor)
		for _, author := range turboAuthors {
			tb := (*TurboAuthor)(author)
			turboAuthorsByID[author.Model.ID] = tb
		}
		return func(book *Book) *TurboAuthor {
			return turboAuthorsByID[safePtr(book.AuthorID)]
		}, nil
	})
}

// func (t *TurboBook) Author(ctx context.Context, db *gorm.DB) (*TurboAuthor, error) {
// 	return turbo.LoadParent(ctx, t.Loader, "author", t.Model, turbo.LoadParentArgs[uint, *Book, *Author, *TurboAuthor]{
// 		ModelParentIDFunc: func(book *TurboBook) *uint { return book.Model.AuthorID },
// 		QueryParentFunc: func(ctx context.Context, ids []uint) ([]*Author, error) {
// 			var authors []*Author
// 			if err := db.WithContext(ctx).Debug().Where("id IN (?)", ids).Find(&authors).Error; err != nil {
// 				return nil, err
// 			}
// 			return authors, nil
// 		},
// 		TurboConstructor: NewTurboAuthors,
// 		ParentIDFunc:     func(author *TurboAuthor) uint { return author.Model.ID },
// 	})
// }

func safePtr[T any](v *T) T {
	var empty T
	if v == nil {
		return empty
	}
	return *v
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

	// var authors []*Author
	// if err := db.Find(&authors).Error; err != nil {
	// 	t.Fatalf("Failed to find authors: %v", err)
	// }

	// turboAuthors := NewTurboAuthors(authors)
	// for _, turboAuthor := range turboAuthors {
	// 	fmt.Println(turboAuthor.Model.Name)
	// 	books, err := turboAuthor.Books(ctx, db)
	// 	if err != nil {
	// 		t.Fatalf("Failed to get books: %v", err)
	// 	}
	// 	for _, book := range books {
	// 		fmt.Println("  ", book.Model.Title)
	// 		chapters, err := book.Chapters(ctx, db)
	// 		if err != nil {
	// 			t.Fatalf("Failed to get chapters: %v", err)
	// 		}
	// 		for _, chapter := range chapters {
	// 			fmt.Println("    ", chapter.Model.Title)
	// 		}
	// 	}
	// }

	var books []*Book
	if err := db.Find(&books).Error; err != nil {
		t.Fatalf("Failed to find books: %v", err)
	}
	turboBooks := NewTurboBooks(books)
	for _, book := range turboBooks {
		author, err := book.Author(ctx, db)
		if err != nil {
			t.Fatalf("Failed to get author: %v", err)
		}
		if author == nil {
			fmt.Println(book.Model.Title, "by <unknown author>")
		} else {
			fmt.Println(book.Model.Title, "by", author.Model.Name)
		}
	}
}
