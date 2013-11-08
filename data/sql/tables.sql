
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

