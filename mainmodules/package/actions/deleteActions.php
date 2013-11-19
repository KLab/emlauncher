<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Random.php';

class deleteActions extends packageActions
{
	const SESKEY_TOKEN = 'package_delete_token';

	protected $package = null;
	protected $app = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(!$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
	}

	public function executeDelete_confirm()
	{
		$token  = Random::string(32);
		mfwSession::set(self::SESKEY_TOKEN,$token);

		$parma = array(
			'token' => $token,
			);
		return $this->build($parma);
	}

	public function executeDelete()
	{
		// todo: token確認

		// DBから削除

		// S3から削除(例外無視)

		$this->redirect("/app?id={$this->app->getId()}");
	}

}

