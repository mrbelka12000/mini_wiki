begin;
alter table files drop column  if exists  project;

commit;
