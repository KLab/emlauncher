<?php

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

//		$this->login_user = User::getLoginUser();
		if(!$this->login_user && $this->getModule()!='login'){
			mfwMemcache::storeUrl(mfwRequest::url());
			return $this->redirect('/login');
		}
	}

}
