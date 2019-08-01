<?php
require_once APP_ROOT.'/model/User.php';

class ajaxActions extends MainActions
{
	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}

		$this->login_user = User::getLoginUser();
		if(!$this->login_user){
			return $this->jsonResponse(
				self::HTTP_403_FORBIDDEN,
				array('error'=>'login required'));
		}

		apache_log('user',$this->login_user->getMail());
		return null;
	}

	public function executeDefaultAction()
	{
		return $this->jsonResponse(
			self::HTTP_404_NOTFOUND,
			array('error'=>'404 Not Found'));
	}
}
