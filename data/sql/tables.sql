
drop table if exists `user_pass`;
create table `user_pass` (
  `mail` varchar(255) not null,
  `passhash` varchar(255) default null,
  primary key (`mail`)
)Engine=InnoDB default charset=utf8;


drop table if exists `application`;
create table `application` (
  `id` integer not null auto_increment,
  `title` varchar(255) not null,
  `api_key` varchar(255) not null,
  `icon_key` varchar(255) default null,
  `description` varchar(1000) default null,
  `repository` varchar(1000) default null,
  `last_updated` datetime default null,
  `created` datetime not null,
  unique key `idx_api_key` (`api_key`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `application_owner`;
create table `application_owner` (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `owner_mail` varchar(255) not null,
  key `idx_app` (`app_id`),
  unique key `idx_owner_app` (`owner_mail`,`app_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `tag`;
create table `tag` (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `name` varchar(255) not null,
  key `idx_app` (`api_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `package`;
create table `package` (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `platform` varchar(31) not null,
  `file_name` varchar(63) not null,
  `title` varchar(255) not null,
  `description` varchar(1000) default null,
  `created` datetime not null,
  key `idx_app` (`api_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `package_tag`;
create table `package_tag` (
  `package_id` integer not null,
  `tag_id` integer not null,
  primary key (`package_id`,`tag_id`)
)Engine=InnoDB default charset=utf8;

