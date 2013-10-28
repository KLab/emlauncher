
create database if not exists `emlauncher`;

drop table if exists `user_pass`;
create table `user_pass` (
  `mail` varchar(256) not null,
  `pass` varchar(256) default null,
  primary key (`mail`)
)Engine=InnoDB default charset=utf8;


