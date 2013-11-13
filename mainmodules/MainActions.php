<?php
require_once APP_ROOT.'/model/User.php';

class MainActions extends mfwActions
{
	const TEMPLATEDIR = '/data/templates';
	const BLOCKDIR = '/data/blocks';

	const SESKEY_URL_BEFORE_LOGIN = 'url_before_login';

	protected $login_user = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}

		// package/install_plist はセッションが使えないためシグネチャで認証する.
		if($this->getModule()==='package' && $this->getAction()==='install_plist'){
			return null;
		}

		$this->login_user = User::getLoginUser();
		if(!$this->login_user && $this->getModule()!='login'){
			$this->saveUrlBeforeLogin();
			return $this->redirect('/login');
		}

		return null;
	}

	protected function build($params,$headers=array())
	{
		$params['login_user'] = $this->login_user;
		return parent::build($params,$headers);
	}


	protected function buildErrorPage($message,$headers=array())
	{
		$params = array(
			'message' => $message,
			);
		$this->setTemplateName('_error');
		return $this->build($params,$headers);
	}

	protected function response($status,$message=null)
	{
		return array(array($status),$message);
	}

	protected function saveUrlBeforeLogin()
	{
		if(mfwRequest::method()==='GET'){
			$url = mfwRequest::url();
			mfwSession::set(self::SESKEY_URL_BEFORE_LOGIN,$url);
		}
		else{
			mfwSession::clear(self::SESKEY_URL_BEFORE_LOGIN);
		}
	}

	protected function redirectUrlBeforeLogin()
	{
		$url = mfwSession::get(self::SESKEY_URL_BEFORE_LOGIN);
		if(!$url){
			$url = '/';
		}
		return $this->redirect($url);
	}

	protected function clearUrlBeforelogin()
	{
		mfwSession::clear(self::SESKEY_URL_BEFORE_LOGIN);
	}

}
