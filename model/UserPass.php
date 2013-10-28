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
	public function sendResetMail($sender)
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
		$from = "From: $sender";

		$body = "EMLauncher password reset URL:\n$url\n";

		if(!mb_send_mail($to,$subject,$body,$from)){
			throw new RuntimeException("mb_send_mail faild (key:$key to:$to)");
		}
	}

	public function updateNewPasshash($password,$con=null)
	{
		
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

