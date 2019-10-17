<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/InstallLog.php';
require_once APP_ROOT.'/model/Random.php';
require_once APP_ROOT.'/model/GuestPass.php';
require_once APP_ROOT.'/model/GuestPassLog.php';

class guestpass_installActions extends packageActions
{
	const TOKEN_KEY_PREFIX = 'ios_plist_token_';

	protected function makeToken()
	{
		$pkg_id = $this->package->getId();
		$token = Random::string(32);
		// tokenは60秒有効
		mfwMemcache::set(self::TOKEN_KEY_PREFIX.$token,$pkg_id,60);
		return $token;
	}
	protected function checkToken($token)
	{
		$pkg_id = mfwMemcache::get(self::TOKEN_KEY_PREFIX.$token);
		return ($pkg_id==$this->package->getId());
	}

	public function executeGuestpass_install()
	{
		$pf = $this->package->getPlatform();
		$ua = mfwRequest::userAgent();
		$remote_ip = mfwRequest::remoteHost();

		if($pf===Package::PF_IOS && $ua->isIOS()){
			// itms-service での接続はセッションを引き継げない
			// 一時トークンをURLパラメータに付けることで認証する
			$scheme = Config::get('enable_https')?'https':null; // HTTPSが使えるならHTTPS優先
			$plist_url = mfwHttp::composeUrl(
				mfwRequest::makeUrl('/package/install_plist',$scheme),
				array(
					'id' => $this->package->getId(),
					't' => $this->makeToken(),
					));
			$url = 'itms-services://?action=download-manifest&url='.urlencode($plist_url);
		}
		else if($ua->isAndroid() && $this->package->isAndroidAppBundle()){
			// AndroidからのアクセスでAABファイルの時、添付ファイルにAPKがあればそれをDL
			$target = $this->package->getAttachedFiles()->pickupByType(AttachedFile::TYPE_APK);
			if(!$target){
				$target = $this->package;
			}
			$url = $target->getFileUrl(self::TIME_LIMIT);
		}
		else{
			// それ以外のアクセスはパッケージを直接DL
			$url = $this->package->getFileUrl(self::TIME_LIMIT);
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			GuestPassLog::Logging($this->guest_pass ,$ua, $remote_ip, $con);
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			throw $e;
		}

		apache_log('app_id',$this->app->getId());
		apache_log('pkg_id',$this->package->getId());
		apache_log('platform',$this->package->getPlatform());

		return $this->redirect($url);
	}

	public function executeInstall_plist()
	{
		$token = mfwRequest::param('t');
		if(!$this->checkToken($token)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}

		$pkg = $this->package;
		$app = $pkg->getApplication();

		$ipa_url = $pkg->getFileUrl('+60 min');
		$image_url = $app->getIconUrl();
		$bundle_identifier = $pkg->getIdentifier();
		$pkg_title = $pkg->getTitle();
		$app_title = $app->getTitle();

		ob_start();
		include APP_ROOT.'/data/templates/package/install_plist.php';
		$plist = ob_get_clean();

		$header = array(
			'Content-Type: text/xml',
			);
		return array($header,$plist);
	}

}

