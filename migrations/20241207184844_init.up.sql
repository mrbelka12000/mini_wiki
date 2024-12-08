create table files
(
    id            serial
        primary key,
    title         text not null,
    text          text,
    file_key      text,
    search_vector tsvector,
    created_at    timestamp default now()
);

create index articles_search_vector_idx
    on files using gin (search_vector);






