alter table package add column `protect` tinyint not null default 0 comment '保護フラグ. 0:自動削除する; 1:自動削除対象外' after file_size;
