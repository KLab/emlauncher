<?php

class loginActions extends MainActions
{
	protected $config;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if($this->login_user){
			// ログイン済みならTOPに飛ばす
			return $this->redirect('/');
		}

		include APP_ROOT.'/config/login_config.php';
		$this->config = $login_config[mfwServerEnv::getEnv()];

		return null;
	}

	protected function build($params=array())
	{
		$params['enable_password'] = $this->config['enable_password'];
		$params['enable_google_auth'] = $this->config['enable_google_auth'];
		return parent::build($params);
	}

	public function executeIndex()
	{
		return $this->build();
	}

}