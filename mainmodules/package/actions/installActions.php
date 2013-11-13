<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/InstallLog.php';

class installActions extends packageActions
{
	public function executeInstall()
	{
		$pf = $this->package->getPlatform();
		$ua = mfwRequest::userAgent();

		if($pf===Package::PF_IOS && $ua->isIOS()){
			// itms-service での接続はセッションを引き継げない
			// シグネチャをURLパラメータに付けることで認証する
			// todo: signe
			$plist_url = mfwRequest::makeUrl("/package/install_plist?id={$this->package->getId()}");
			$url = 'itms-services://?action=download-manifest&url='.urlencode($plist_url);
		}
		else{
			// iPhone以外でのアクセスはパッケージを直接DL
			$url = $this->package->getFileUrl('+5 min');
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			InstallLog::Logging($this->login_user,$this->package,$ua,$con);
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log(__METHOD__.': '.$e->getMessage());
			throw $e;
		}

		return $this->redirect($url);
	}

	public function executeInstall_plist()
	{
		// todo: check signature

		$pkg = $this->package;
		$app = $pkg->getApplication();

		$ipa_url = $pkg->getFileUrl('+5 min');
		$image_url = $app->getIconUrl();
		$bundle_identifier = $pkg->getIOSIdentifier();
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

