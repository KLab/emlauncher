<?php

/**
 */
class InstallLog {

	public static function Logging(User $user,Package $pkg,mfwUserAgent $ua,$con=null)
	{
		$sql = 'INSERT INTO install_log'
			. ' (app_id,package_id,mail,user_agent,installed)'
			. ' VALUES (:app_id,:package_id,:mail,:user_agent,:installed)';
		$bind = array(
			':app_id' => $pkg->getAppId(),
			':package_id' => $pkg->getId(),
			':mail' => $user->getMail(),
			':user_agent' => $ua->getString(),
			':installed' => date('Y-m-d H:i:s'),
			);
		mfwDBIBase::query($sql,$bind,$con);

		$sql = 'INSERT IGNORE INTO app_install_user (app_id,mail) VALUES (:app_id,:mail)';
		$bind = array(
			':app_id' => $pkg->getAppId(),
			':mail' => $user->getMail(),
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

	public static function getPackageInstallCount(Package $pkg)
	{
		$sql = 'SELECT count(distinct(mail)) FROM install_log WHERE package_id = ?';
		return (int)mfwDBIBase::getOne($sql,array($pkg->getId()));
	}

	public static function getApplicationInstallUserCount(Application $app)
	{
		$sql = 'SELECT count(*) FROM app_install_user WHERE app_id = ?';
		return (int)mfwDBIBase::getOne($sql,array($app->getId()));
	}

}

