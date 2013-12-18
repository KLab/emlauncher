<?php
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/Random.php';

class packageActions extends MainActions
{
	const INSTALL_TOKEN_PREFIX = 'pkg_install_token_';

	protected $package = null;
	protected $app = null;

	public function initialize()
	{
		if($this->action==='install' && mfwRequest::has('token')){
			// token付きインストールリンクの場合, token情報のみで認証する.
			return $this->initializeByInstallToken(mfwRequest::param('token'));
		}

		if(($err=parent::initialize())){
			return $err;
		}
		$id = mfwRequest::param('id');
		$this->package = PackageDb::retrieveByPK($id);
		if(!$this->package){
			return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
		}
		$this->app = $this->package->getApplication();
		return null;
	}

	public function build($params)
	{
		$params['package'] = $this->package;
		$params['app'] = $this->app;
		return parent::build($params);
	}


	public function executeIndex()
	{
		$params = array(
			);
		return $this->build($params);
	}


	/**
	 * インストールトークンを使って初期化.
	 * parentのinitializeは呼ばず、ここで認証と初期化を済ませる.
	 */
	public function initializeByInstallToken($token)
	{
		$tokendata = $this->getTokenData($token);
		apache_log('token_data',$tokendata);

		if(!$tokendata){
			error_log("invalid install token: $token");
			return $this->response(self::HTTP_403_FORBIDDEN,'invalid token');
		}

		$this->package = PackageDb::retrieveByPK($tokendata['package_id']);
		if(!$this->package){
			return $this->response(self::HTTP_404_NOTFOUND,'');
		}

		$this->app = $this->package->getApplication();
		$this->login_user = new User($tokendata['mail']);

		return null;
	}

	protected function getTokenData($token)
	{
		$tokendata = mfwMemcache::get(self::INSTALL_TOKEN_PREFIX.$token);
		$tokendata = json_decode($tokendata,true);
		if(!$tokendata){
			return null;
		}
		if(strtotime($tokendata['expire']) < time()){
			return null;
		}
		return $tokendata;
	}

	public function executeCreate_token()
	{
		$token_expire = '+1 hours';

		$expire_time = strtotime($token_expire);
		$mc_expire = $expire_time - time();

		$tokendata = array(
			'mail' => $this->login_user->getMail(),
			'package_id' => $this->package->getId(),
			'expire' => date('Y-m-d H:i:s',$expire_time),
			);
		$token = Random::string(32);
		mfwMemcache::set(self::INSTALL_TOKEN_PREFIX.$token,json_encode($tokendata),$mc_expire);

		apache_log('token',$token);
		apache_log('token_data',$tokendata);

		$params = array(
			'token' => $token,
			'expire' => $tokendata['expire'],
			'token_url' => mfwRequest::makeURL("/package/install?token={$token}"),
			);
		return $this->build($params);
	}

}
