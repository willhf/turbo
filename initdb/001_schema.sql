create schema if not exists app;

create table if not exists app.authors (
  id bigserial primary key,
  name text not null
);

create table if not exists app.books (
  id bigserial primary key,
  author_id bigint references app.authors(id) on delete cascade,
  title text not null
);

create table if not exists app.chapters (
  id bigserial primary key,
  book_id bigint references app.books(id) on delete cascade,
  title text not null
);

create index if not exists idx_books_author_id on app.books(author_id);
create index if not exists idx_chapters_book_id on app.chapters(book_id);