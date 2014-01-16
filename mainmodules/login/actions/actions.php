<?php
require_once APP_ROOT.'/model/Config.php';

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

		$this->config = Config::get('login');

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
		// httpsが使えるならlogin画面はhttpsを強制
		if(Config::get('enable_https')){
			if(strpos(mfwRequest::url(),'http://')===0){
				return $this->redirect(mfwrequest::makeUrl('/login','https'));
			}
		}

		return $this->build();
	}

}