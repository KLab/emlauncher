<?php
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/Random.php';
require_once APP_ROOT.'/model/GuestPass.php';
require_once APP_ROOT.'/model/GuestPassLog.php';

class packageActions extends MainActions
{
	const INSTALL_TOKEN_PREFIX = 'pkg_install_token_';
	const TIME_LIMIT = '+60 min';

	/**
	 * @var Package
	 */
	protected $package = null;

	/**
	 * @var Application
	 */
	protected $app = null;

	/**
	 * @var GuestPass
	 */
	protected $guest_pass = null;

	public function initialize()
	{
		if($this->action==='install' && mfwRequest::has('token')){
			// token付きインストールリンクの場合, token情報のみで認証する.
			return $this->initializeByInstallToken(mfwRequest::param('token'));
		}

		if($this->action==='guestpass_install' && mfwRequest::has('token')){
			// guestpass installの場合, guestpass token情報のみで認証する.
			return $this->initializeByGuestPassToken(mfwRequest::param('token'));
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

	public function build($params=array(),$headers=array())
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

	public function initializeByGuestPassToken($token)
	{
		$this->guest_pass = GuestpassDb::selectByToken($token);
		apache_log('guest_pass', $this->guest_pass);
		if (is_null($this->guest_pass)) {
			return $this->BuildErrorPage('Not Found', array(self::HTTP_404_NOTFOUND));
		}
		if (strtotime($this->guest_pass->getExpired()) < time()) {
			error_log("expired guestpass: $token (app:{$this->guest_pass->getAppId()} package:{$this->guest_pass->getPackageId()} mail:{$this->guest_pass->getMail()})");
			return $this->BuildErrorpage('Invalid token', array(self::HTTP_403_FORBIDDEN));
		}

		$this->package = PackageDb::retrieveByPK($this->guest_pass->getPackageId());

		if (!$this->package) {
			return $this->response(self::HTTP_404_NOTFOUND, '');
		}

		$this->app = $this->package->getApplication();
		$this->login_user = new User($this->guest_pass->getMail());

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

	public function executeCreate_guestpass()
	{
		// オーナー以外はguestpass作成出来ない
		if ($this->app->isOwner($this->login_user) == false) {
			return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
		}
		$expired = "+1 week";

		$expire_time = strtotime($expired);

		$tokendata = array(
			'mail' => $this->login_user->getMail(),
			'package_id' => $this->package->getId(),
			'expire' => date('Y-m-d H:i:s', $expire_time),
		);
		$token = Random::string(32);

		// TODO guest_passをDBに保存
		$guest_pass = GuestPassDb::insertNewGuestPass(
			$this->package,
			$this->login_user,
			$token,
			$tokendata['expire']
		);

		return $this->redirect("/package/guestpass?id={$this->package->getId()}&guestpass_id={$guest_pass->getId()}");
	}

	public function executeGuestpass()
	{
		/* @var GuestPass $guest_pass */
		$guest_pass = GuestPassDb::retrieveByPK(mfwRequest::param('guestpass_id'));

		if (is_null($guest_pass) || $this->app->isOwner($this->login_user) == false) {
			return $this->buildErrorPage('Forbidden',array(self::HTTP_403_FORBIDDEN));
		}

		$params = array(
			'token' => $guest_pass->getToken(),
			'expire' => $guest_pass->getExpired(),
			'guestpass_url' => mfwRequest::makeUrl("/package/guestpass_install?token={$guest_pass->getToken()}"),
		);
		return $this->build($params);
	}

	public function executeExpire_guestpass()
	{
		/* @var GuestPass $guest_pass */
		$guest_pass = GuestPassDb::retrieveByPK(mfwRequest::param('guestpass_id'));

		if (is_null($guest_pass) || $this->app->isOwner($this->login_user) == false) {
			return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
		}

		$guest_pass->execExpired();

		return $this->redirect("/package/?id={$this->package->getId()}");
	}
}
