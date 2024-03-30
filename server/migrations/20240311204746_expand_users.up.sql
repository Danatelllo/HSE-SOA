CREATE TABLE users (
  id bigserial not null primary key,
  user_email varchar,
  user_name varchar,
  user_login varchar not null unique,
  user_surname varchar,
  user_phone_number varchar,
  user_password varchar,
  user_birthday date
);
