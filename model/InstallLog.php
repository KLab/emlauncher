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

}

