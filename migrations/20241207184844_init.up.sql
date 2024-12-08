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

create table file_names
(
    file_key text unique not null,
    count int not null
);


CREATE INDEX file_names_idx on file_names(file_key);

create index files_search_vector_idx
    on files using gin (search_vector);







