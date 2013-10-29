<?php
require_once APP_ROOT.'/model/User.php';

class MainActions extends mfwActions
{
	const TEMPLATEDIR = '/data/templates';
	const BLOCKDIR = '/data/blocks';

	protected $login_user = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}

		$this->login_user = User::getLoginUser();
		if(!$this->login_user && $this->getModule()!='login'){
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

}
