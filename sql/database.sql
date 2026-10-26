-- DB作成sql.
-- パスワードは適宜変更して, 設定ファイルの database (user / password または authfile) に反映ください
create database `emlauncher`;
grant all on emlauncher.* to 'emlauncher'@'localhost' identified by 'xxxxxxxx'; 
