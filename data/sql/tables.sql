
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
  `api_key` varchar(255) not null comment 'Upload APIアクセス用のキー',
  `icon_key` varchar(255) default null comment 'S3のアイコンファイルのキー',
  `description` varchar(1000) default null,
  `repository` varchar(1000) default null comment 'リポジトリURLなど',
  `last_upload` datetime default null comment 'パッケージの最終アップロード時刻',
  `last_commented` datetime default null comment '最終コメント時刻',
  `created` datetime not null,
  `date_to_sort` datetime not null comment 'last_upload,last_comment,createdのうち最新のもの',
  unique key `idx_api_key` (`api_key`),
  key idx_date_to_sort (`date_to_sort`) comment '新着ソート用',
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
  key `idx_app` (`app_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `package`;
create table `package` (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `platform` varchar(31) not null comment '"Android","iOS","unknown"',
  `file_name` varchar(63) not null comment 'ベースファイル名. app_idやidを加えてS3のキーを作る.',
  `title` varchar(255) not null,
  `description` text,
  `ios_identifier` varchar(255) default null comment 'ipaのCFBundleIdentifier',
  `original_file_name` varchar(255) default null,
  `file_size` integer default null,
  `created` datetime not null,
  key `idx_app` (`app_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8 comment 'インストールパッケージ';

drop table if exists `package_tag`;
create table `package_tag` (
  `package_id` integer not null,
  `tag_id` integer not null,
  key `idx_tag` (`tag_id`),
  primary key (`package_id`,`tag_id`)
)Engine=InnoDB default charset=utf8 comment 'packageとtagのjunction';

drop table if exists `install_log`;
create table install_log (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `package_id` integer not null,
  `mail` varchar(255) not null,
  `user_agent` varchar(1000) not null,
  `installed` datetime not null comment 'インストール日時',
  key idx_mail_app (`mail`,`app_id`,`package_id`),
  key idx_package (`package_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `app_install_user`;
create table app_install_user (
  `app_id` integer not null,
  `mail` varchar(255) not null,
  `notify` tinyint not null default 1 comment '更新通知設定. 0:送る; 1:送らない',
  `last_installed` datetime not null,
  key idx_app (`app_id`),
  primary key (`mail`,`app_id`)
)Engine=InnoDB default charset=utf8;

drop table if exists `comment`;
create table `comment` (
  `id` integer not null auto_increment,
  `app_id` integer not null,
  `package_id` integer default null,
  `number` integer not null comment 'アプリ毎の通し番号',
  `mail` varchar(255) not null comment 'コメントした人',
  `message` text not null,
  `created` datetime not null,
  key idx_app (`app_id`),
  key idx_pkg (`package_id`),
  primary key (`id`)
)Engine=InnoDB default charset=utf8;

