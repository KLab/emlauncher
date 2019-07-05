<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/Random.php';

class deleteActions extends packageActions
{
	const SESKEY_TOKEN = 'package_delete_token';

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(!$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
	}

	public function executeDelete_confirm()
	{
		$token  = Random::string(32);
		mfwSession::set(self::SESKEY_TOKEN,$token);

		$parma = array(
			'token' => $token,
			);
		return $this->build($parma);
	}

	public function executeDelete()
	{
		$token = mfwRequest::param('token',null,'POST');

		if($token!==mfwSession::get(self::SESKEY_TOKEN)){
			return $this->buildErrorPage(
				'Bad Request',array(self::HTTP_400_BADREQUEST));
		}
		mfwSession::clear(self::SESKEY_TOKEN);

		apache_log('app_id',$this->app->getId());
		apache_log('pkg_id',$this->package->getId());
		apache_log('platform',$this->package->getPlatform());
		apache_log('attached_ids',$this->package->getAttachedFiles()->getColumnArray('id'));

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$this->app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

			$this->package->delete($con);

			if($this->app->getLastUpload()==$this->package->getCreated()){
				// 最終アップデート時刻を前のものに戻す
				$pkg = PackageDb::selectNewestOneByAppId($this->app->getId());
				$lastupload = ($pkg)? $pkg->getCreated(): null;

				$this->app->updateLastUpload($lastupload,$con);
			}

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			throw $e;
		}

		try{
			$this->package->deleteFile();
			$this->package->getAttachedFiles()->deleteFiles();
		}
		catch(Exception $e){
			// S3から削除出来なくてもDBからは消えているので許容する
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
		}

		return $this->redirect("/app?id={$this->app->getId()}");
	}

}

