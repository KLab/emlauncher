<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/UserPass.php';

class passwordActions extends loginActions
{

	public function executePassword()
	{
		$mail = mfwRequest::param('email');
		$pass = mfwRequest::param('password');

		$user_pass = UserPassDb::selectByEmail($mail);
		if(!$user_pass || !$user_pass->checkPassword($pass)){
			return $this->buildErrorPage('invalid email or password');
		}

		User::login($mail);

		return $this->redirectUrlBeforeLogin();
	}

	public function executePassword_reminder()
	{
		return $this->build();
	}

	public function executePassword_confirm()
	{
		$params = array('error'=>0);

		$mail = mfwRequest::param('email');
		$user_pass = UserPassDb::selectByEmail($mail);
		if($user_pass){
			$user_pass->sendResetMail();
		}
		else{
			$params['error'] = 1;
		}
		return $this->build($params);
	}


	public function executePassword_reset()
	{
		$key = mfwRequest::param('key');
		$data = mfwMemcache::get($key);
		if(!$data){
			return $this->buildErrorPage('invalid key');
		}
		$params = array(
			'key' => $key,
			'data' => $data,
			);
		return $this->build($params);
	}

	public function executePassword_commit()
	{
		$key = mfwRequest::param('key');
		$pass = mfwRequest::param('password');

		$data = mfwMemcache::get($key);
		$user_pass = null;
		if(isset($data['mail'])){
			$user_pass = UserPassDb::selectByEmail($data['mail']);
		}
		if(!$user_pass){
			return $this->buildErrorPage('invalid key');
		}

		$user_pass->updatePasshash($pass);
		mfwMemcache::delete($key);

		return $this->build();
	}

}
