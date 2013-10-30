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

		$this->login_user = User::getLoginUser();
		if(!$this->login_user && $this->getModule()!='login'){
			$this->saveUrlBeforeLogin();
			return $this->redirect('/login');
		}

		return null;
	}

	protected function build($params)
	{
		$params['login_user'] = $this->login_user;
		return parent::build($params);
	}


	protected function buildErrorPage($message)
	{
		$params = array(
			'message' => $message,
			);
		$this->setTemplateName('_error');
		return $this->build($params);
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
