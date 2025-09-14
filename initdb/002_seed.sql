insert into app.authors (name) values
  ('Alice Pennington'),   -- will get 2 books
  ('Marcus Vellum'),      -- will get 1 book
  ('Sofia Duarte'),       -- will get 1 book
  ('Jamal Whitaker'),     -- will get 0 books
  ('Harper Lin');         -- will get 0 books

insert into app.books (author_id, title)
select id, 'The Clockwork Harbor'
from app.authors where name = 'Alice Pennington';

insert into app.books (author_id, title)
select id, 'Shadows in Amber'
from app.authors where name = 'Alice Pennington';

insert into app.books (author_id, title)
select id, 'Songs of the Ironwood'
from app.authors where name = 'Marcus Vellum';

insert into app.books (author_id, title)
select id, 'The Long Voyage North'
from app.authors where name = 'Sofia Duarte';

insert into app.books (author_id, title)
values (null, 'Orphaned Manuscript');
