create table if not exists book
(
    id               varchar(36)   not null
        primary key,
    owner_id         varchar(255)  null,
    name             varchar(50)   not null,
    description      text          not null,
    v_ip_level       varchar(255)  null,
    price            decimal       null,
    cover_image      varchar(255)  not null,
    browse_amount    int default 0 not null,
    bookshelf_amount int default 0 not null,
    pushed_at        timestamp     not null,
    thermal_value    int default 0 not null,
    created_at       datetime(3)   null,
    updated_at       datetime(3)   null,
    deleted_at       datetime(3)   null,
    constraint idx_book_name
        unique (name)
);

create index idx_book_deleted_at
    on book (deleted_at);

create table if not exists book_chapter
(
    id         varchar(36)   not null
        primary key,
    book_id    varchar(36)   null,
    title      varchar(255)  null,
    content    text          null,
    `order`    int default 1 not null,
    v_ip_level varchar(255)  null,
    price      decimal       null,
    created_at datetime(3)   null,
    updated_at datetime(3)   null,
    deleted_at datetime(3)   null
);

create index idx_book_chapter_deleted_at
    on book_chapter (deleted_at);

create table if not exists book_tag
(
    id         varchar(36)  not null
        primary key,
    book_id    varchar(36)  null,
    tag        varchar(255) null,
    created_at datetime(3)  null
);

create index idx_book_tag_book_id
    on book_tag (book_id);

create table if not exists `order`
(
    id             varchar(36)    not null
        primary key,
    user_id        varchar(36)    null,
    pay_price      decimal(10, 2) null,
    original_price decimal(10, 2) null,
    reword_gold    decimal        null,
    past_at        datetime(3)    null,
    status         tinyint        null,
    created_at     datetime(3)    null,
    updated_at     datetime(3)    null,
    deleted_at     datetime(3)    null,
    paid_at        datetime(3)    null
);

create index idx_order_deleted_at
    on `order` (deleted_at);

create table if not exists outbox
(
    id         varchar(36)  not null
        primary key,
    type       varchar(255) null,
    operation  varchar(255) null,
    payload    json         null,
    status     varchar(255) null,
    created_at datetime(3)  null,
    updated_at datetime(3)  null
);

create table if not exists user
(
    id            varchar(36)                not null
        primary key,
    username      varchar(50)                not null,
    email         varchar(100)               not null,
    password      varchar(255)               not null,
    age           bigint                     null,
    description   text                       null,
    profile_image varchar(255)               null,
    cover_image   varchar(255)               null,
    role          varchar(20) default 'user' null,
    created_at    datetime(3)                null,
    updated_at    datetime(3)                null,
    deleted_at    datetime(3)                null,
    constraint idx_user_email
        unique (email),
    constraint idx_user_username
        unique (username)
);

create index idx_user_deleted_at
    on user (deleted_at);

create table if not exists user_asset
(
    id             varchar(36) not null
        primary key,
    gold           int         not null,
    v_ip_level     varchar(20) not null,
    v_ip_active_at datetime(3) null,
    created_at     datetime(3) null,
    updated_at     datetime(3) null,
    deleted_at     datetime(3) null
);

create index idx_user_asset_deleted_at
    on user_asset (deleted_at);

create table if not exists user_bookshelf
(
    id         varchar(36)   not null
        primary key,
    user_id    varchar(36)   null,
    book_id    varchar(36)   null,
    `order`    int default 1 not null,
    created_at datetime(3)   null,
    updated_at datetime(3)   null,
    deleted_at datetime(3)   null
);

create index idx_user_bookshelf_book_id
    on user_bookshelf (book_id);

create index idx_user_bookshelf_user_id
    on user_bookshelf (user_id);

create table if not exists user_history
(
    id         varchar(36) not null
        primary key,
    user_id    varchar(36) not null,
    book_id    varchar(36) not null,
    created_at datetime(3) null,
    updated_at datetime(3) null,
    deleted_at datetime(3) null
);

create index idx_user_history_deleted_at
    on user_history (deleted_at);

create table if not exists user_purchased_book
(
    id         varchar(36) not null
        primary key,
    user_id    varchar(36) null,
    book_id    varchar(36) null,
    active_at  datetime(3) null,
    created_at datetime(3) null,
    updated_at datetime(3) null,
    deleted_at datetime(3) null
);

create index idx_user_purchased_book_active_at
    on user_purchased_book (active_at);

create index idx_user_purchased_book_book_id
    on user_purchased_book (book_id);

create index idx_user_purchased_book_created_at
    on user_purchased_book (created_at);

create index idx_user_purchased_book_deleted_at
    on user_purchased_book (deleted_at);

create index idx_user_purchased_book_updated_at
    on user_purchased_book (updated_at);

create index idx_user_purchased_book_user_id
    on user_purchased_book (user_id);

create table if not exists user_purchased_chapter
(
    id         varchar(36) not null
        primary key,
    user_id    varchar(36) null,
    chapter_id varchar(36) null,
    book_id    varchar(36) null,
    active_at  datetime(3) null,
    created_at datetime(3) null,
    updated_at datetime(3) null,
    deleted_at datetime(3) null
);

create index idx_user_purchased_chapter_active_at
    on user_purchased_chapter (active_at);

create index idx_user_purchased_chapter_book_id
    on user_purchased_chapter (book_id);

create index idx_user_purchased_chapter_chapter_id
    on user_purchased_chapter (chapter_id);

create index idx_user_purchased_chapter_created_at
    on user_purchased_chapter (created_at);

create index idx_user_purchased_chapter_deleted_at
    on user_purchased_chapter (deleted_at);

create index idx_user_purchased_chapter_updated_at
    on user_purchased_chapter (updated_at);

create index idx_user_purchased_chapter_user_id
    on user_purchased_chapter (user_id);


