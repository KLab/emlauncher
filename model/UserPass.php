<?php

/**
 * Row object for 'user_pass' table.
 */
class UserPass extends mfwObject {
	const DB_CLASS = 'UserPassDb';
	const SET_CLASS = 'UserPassSet';

	const RESET_MAIL_EXPIRE = 1800;

	/**
	 * @param[in] string $sender 送信アドレス
	 */
	public function sendResetMail()
	{
		$data = array(
			'mail' => $this->row['mail'],
			'microtime' => microtime(),
			);
		$key = sha1(json_encode($data));
		mfwMemcache::set($key,$data,self::RESET_MAIL_EXPIRE);

		$url = mfwRequest::makeUrl("/login/password_reset?key=$key");

		$subject = 'Reset password';
		$to = $data['mail'];
		$from = 'From: '.Config::get('mail_sender');

		$body = "EMLauncher password reset URL:\n$url\n";

		if(!mb_send_mail($to,$subject,$body,$from)){
			throw new RuntimeException("mb_send_mail faild (key:$key to:$to)");
		}
	}

	protected function randomstring($length)
	{
		$chars = '1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
		$strlen = strlen($chars);
		$str = '';
		for($i=0;$i<$length;++$i){
			$str .= $chars[mt_rand(0,$strlen-1)];
		}
		return $str;
	}
	protected function calchash($pass,$salt,$stretch)
	{
		$hash = sha1("{$pass}{$salt}");
		for($i=0;$i<$stretch;++$i){
			$hash = sha1("{$hash}{$salt}");
		}
		return $hash;
	}

	public function updatePasshash($password,$con=null)
	{
		$stretch = mt_rand(10,20);
		$salt = $this->randomstring(16);
		$hash = $this->calchash($password,$salt,$stretch);

		$this->row['passhash'] = "{$stretch}:{$salt}:{$hash}";

		$this->update($con);
	}

	public function checkPassword($password)
	{
		$passhash = $this->value('passhash');
		if(!$password || !$passhash){
			return false;
		}
		list($stretch,$salt,$hash) = explode(':',$passhash,3);
		$calc = $this->calchash($password,$salt,$stretch);
		return ($calc==$hash);
	}
}

/**
 * Set of UserPass objects.
 */
class UserPassSet extends mfwObjectSet {
	const PRIMARY_KEY = 'mail';

	public static function hypostatize(Array $row=array())
	{
		return new UserPass($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'user_pass' table.
 */
class UserPassDb extends mfwObjectDb {
	const TABLE_NAME = 'user_pass';
	const SET_CLASS = 'UserPassSet';

	public static function selectByEmail($email)
	{
		return static::selectOne('WHERE mail = ?',array($email));
	}

}

