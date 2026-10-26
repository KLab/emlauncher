-- change charset to utf8mb4 from utf8.

alter table `user_pass`
  modify `mail` varchar(255) character set utf8mb4 not null,
  modify `passhash` varchar(255) character set utf8mb4 default null,
  default charset=utf8mb4;

alter table `application`
  modify `title` varchar(255) character set utf8mb4 not null,
  modify `api_key` varchar(255) character set utf8mb4 not null comment 'Upload APIアクセス用のキー',
  modify `icon_key` varchar(255) character set utf8mb4 default null comment 'S3のアイコンファイルのキー',
  modify `description` varchar(1000) character set utf8mb4 default null,
  modify `repository` varchar(1000) character set utf8mb4 default null comment 'リポジトリURLなど',
  default charset=utf8mb4;

alter table `application_owner`
  modify `owner_mail` varchar(255) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `package_tag`
  default charset=utf8mb4;

alter table `tag`
  modify `name` varchar(255) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `package`
  modify `platform` varchar(31) character set utf8mb4 not null comment '"Android","iOS","unknown"',
  modify `file_name` varchar(63) character set utf8mb4 not null comment 'ベースファイル名. app_idやidを加えてS3のキーを作る.',
  modify `title` varchar(255) character set utf8mb4 not null,
  modify `description` text character set utf8mb4,
  modify `identifier` varchar(255) character set utf8mb4 default null comment 'CFBundleIdentifier/PackageName',
  modify `original_file_name` varchar(255) character set utf8mb4 default null,
  default charset=utf8mb4;

alter table `install_log`
  modify `mail` varchar(255) character set utf8mb4 not null,
  modify `user_agent` varchar(1000) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `app_install_user`
  modify `mail` varchar(255) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `comment`
  modify `mail` varchar(255) character set utf8mb4 not null comment 'コメントした人',
  modify `message` text character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `guest_pass`
  modify `mail` varchar(255) character set utf8mb4 not null,
  modify `token` varchar(32) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `guestpass_log`
  modify `user_agent` varchar(1000) character set utf8mb4 not null,
  modify `ip_address` varchar(255) character set utf8mb4 not null,
  default charset=utf8mb4;

alter table `attached_file`
  modify `file_name` varchar(255) character set utf8mb4 not null,
  modify `original_file_name` varchar(255) character set utf8mb4 not null,
  modify `file_type` varchar(5) character set utf8mb4 not null,
  default charset=utf8mb4;
