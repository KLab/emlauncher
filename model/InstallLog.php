<?php
require_once __DIR__.'/InstallUser.php';
require_once __DIR__.'/InstallApp.php';

/**
 */
class InstallLog {

	public static function Logging(User $user,Package $pkg,mfwUserAgent $ua,$con=null)
	{
		$now = date('Y-m-d H:i:s');
		$sql = 'INSERT INTO install_log'
			. ' (app_id,package_id,mail,user_agent,installed)'
			. ' VALUES (:app_id,:package_id,:mail,:user_agent,:installed)';
		$bind = array(
			':app_id' => $pkg->getAppId(),
			':package_id' => $pkg->getId(),
			':mail' => $user->getMail(),
			':user_agent' => $ua->getString(),
			':installed' => $now,
			);
		mfwDBIBase::query($sql,$bind,$con);

		$sql = 'INSERT INTO app_install_user (app_id,mail,last_installed) VALUES (:app_id,:mail,:last_installed) ON DUPLICATE KEY UPDATE last_installed=:last_installed';
		$bind = array(
			':app_id' => $pkg->getAppId(),
			':mail' => $user->getMail(),
			':last_installed' => $now,
			);
		mfwDBIBase::query($sql,$bind,$con);
	}

	/**
	 * 特定Userの特定Applicationのパッケージのインストール日時.
	 * @return array package_id => datetime string.
	 */
	public static function packageInstalledDates(User $user,$app_id)
	{
		$sql = 'SELECT package_id,max(installed) as installed FROM install_log WHERE app_id=:app_id AND mail = :mail GROUP BY package_id';
		$bind = array(
			':app_id' => $app_id,
			':mail' => $user->getMail(),
			);
		$rows = mfwDBIBase::getAll($sql,$bind);

		$ret = array();
		foreach($rows as $r){
			$ret[$r['package_id']] = $r['installed'];
		}
		return $ret;
	}

	public static function getPackageInstallUsers(Package $pkg)
	{
		$sql = 'SELECT distinct(mail) FROM install_log WHERE package_id = ?';
		$rows = mfwDBIBase::getAll($sql,array($pkg->getId()));
		$mails = array();
		foreach($rows as $r){
			$mails[] = $r['mail'];
		}
		return $mails;
	}

	public static function getInstallApps(User $user)
	{
		$sql = 'SELECT * FROM app_install_user WHERE mail = ?';
		$rows = mfwDBIBase::getAll($sql,array($user->getMail()));
		return new InstallAppSet($user,$rows);
	}

	public static function getInstallApp(User $user,$app_id)
	{
		$sql = 'SELECT * FROM app_install_user WHERE mail = :mail AND app_id = :app_id';
		$bind = array(
			':mail' => $user->getMail(),
			':app_id' => $app_id,
			);
		$row = mfwDBIBase::getRow($sql,$bind);
		if(!$row){
			return null;
		}
		return new InstallApp($row);
	}

	public static function getInstallUsers(Application $app)
	{
		$sql = 'SELECT * FROM app_install_user WHERE app_id = ?';
		$rows = mfwDBIBase::getAll($sql,array($app->getId()));
		return new InstallUserSet($app,$rows);
	}

	public static function getInstallPackageIds(User $user,$app_id)
	{
		$sql = 'SELECT distinct(package_id) as pkg_id FROM install_log WHERE mail=:mail AND app_id=:app_id';
		$bind = array(
			':mail' => $user->getMail(),
			':app_id' => $app_id,
			);
		$rows = mfwDBIBase::getAll($sql,$bind);
		$ids = array();
		foreach($rows as $r){
			$ids[] = $r['pkg_id'];
		}
		return $ids;
	}

}

