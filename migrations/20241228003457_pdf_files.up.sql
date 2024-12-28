begin;

create table pdf_files
(
    id            serial
        primary key,
    title         text not null,
    file_key      text,
    created_at    timestamp default now()
);


commit;