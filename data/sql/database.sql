-- DB作成sql.
-- パスワードは適宜変更して $mfw_server_env の authfile に反映ください
create database `emlauncher`;
grant all on emlauncher.* to 'emlauncher'@'localhost' identified by 'xxxxxxxx'; 
