<?php
require_once APP_ROOT.'/model/InstallLog.php';

class User
{
	const SESKEY = 'login_user';

	protected $mail;
	protected $pkg_install_dates = array();

	public function __construct($mail)
	{
		$this->mail = $mail;
	}

	public function getMail()
	{
		return $this->mail;
	}


	public static function getLoginUser()
	{
		$session = mfwSession::get(self::SESKEY);
		if(!isset($session['mail'])){
			return null;
		}
		return new self($session['mail']);
	}

	public static function login($mail)
	{
		$data = array(
			'mail' => $mail,
			);
		mfwSession::set(self::SESKEY,$data);
		return new self($mail);
	}

	public static function logout()
	{
		mfwSession::clear(self::SESKEY);
	}

	public function getPackageInstalledDate(Package $pkg,$format=null)
	{
		$appid = $pkg->getAppId();
		if(!isset($this->pkg_install_dates[$appid])){
			$this->pkg_install_dates[$appid] = InstallLog::packageInstalledDates($this,$appid);
		}
		if(!isset($this->pkg_install_dates[$appid][$pkg->getId()])){
			return null;
		}
		$date = $this->pkg_install_dates[$appid][$pkg->getId()];
		if($format){
			$date = date($format,strtotime($date));
		}
		return $date;
	}

}

