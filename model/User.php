<?php

class User
{
	const SESKEY = 'login_user';

	protected $mail;

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

}

