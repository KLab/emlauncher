
drop table if exists `user_pass`;
create table `user_pass` (
  `mail` varchar(255) not null,
  `pass` varchar(255) default null,
  primary key (`mail`)
)Engine=InnoDB default charset=utf8;


