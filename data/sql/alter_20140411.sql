
alter table application
  modify column last_upload datetime default null comment 'パッケージの最終アップロード時刻',
  add column last_commented datetime default null comment '最終コメント時刻' after last_upload,
  add column date_to_sort datetime not null comment 'last_upload,last_comment,createdのうち最新のもの' after created,
  drop key idx_last_upload,
  add key idx_date_to_sort (date_to_sort);

update application set last_upload=null;
update application a set last_upload = (select max(created) from package where app_id=a.id);
update application a set last_commented = (select max(created) from comment where app_id=a.id);

update application set date_to_sort = created;
update application set date_to_sort = if(date_to_sort<last_upload, last_upload, date_to_sort) where last_upload is not null;
update application set date_to_sort = if(date_to_sort<last_commented, last_commented, date_to_sort) where last_commented is not null;

