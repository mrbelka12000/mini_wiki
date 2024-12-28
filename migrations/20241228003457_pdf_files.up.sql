begin;

create table pdf_files
(
    id            serial
        primary key,
    title         text not null,
    file_key      text,
    search_data   text,
    created_at    timestamp default now()
);

commit;