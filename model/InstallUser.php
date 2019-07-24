<?php

/*! @file
 * あるアプリをインストールしているユーザの情報.
 * テーブルはapp_install_user.
 * @sa InstallLog
 */

class InstallUser extends mfwObject {
	const SET_CLASS = 'InstallUserSet';

	public function getAppId(){
		return $this->value('app_id');
	}
	public function getMail(){
		return $this->value('mail');
	}
	public function getLastInstalled(){
		return $this->value('last_installed');
	}
	public function getNotifySetting(){
		return (bool)$this->value('notify');
	}
}

class InstallUserSet extends mfwObjectSet {
	const PRIMARY_KEY = 'mail';
	protected $app;

	public function __construct(Application $app,Array $rows=array())
	{
		parent::__construct($rows);
		$this->app = $app;
	}

	public static function hypostatize(Array $row=array())
	{
		return new InstallUser($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}

	public function noticePackageUploaded(Package $pkg)
	{
		$app = $this->app;
		$package_url = mfwRequest::makeURL("/package?id={$pkg->getId()}");
		ob_start();
		include APP_ROOT.'/data/notice_mail_template.php';
		$body = ob_get_clean();

		$addresses = array();
		foreach($this->rows as $r){
			if($r['notify']){
				$addresses[] = $r['mail'];
			}
		}
		if(empty($addresses)){
			return;
		}

		$subject = "New Package Uploaded to {$app->getTitle()}";
		$sender = Config::get('mail_sender');
		$to = Config::get('mail_bcc_to') ?: $sender;
		$header = "From: $sender"
			. "\nBcc: " . implode(', ',$addresses);

		mb_language('uni');
		mb_internal_encoding('UTF-8');
		if(!mb_send_mail($to,$subject,$body,$header)){
			throw new RuntimeException("mb_send_mail faild (pkg={$pkg->getId()}, {$pkg->getTitle()})");
		}
	}
}

