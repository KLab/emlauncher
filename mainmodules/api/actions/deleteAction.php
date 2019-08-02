<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

class deleteAction extends apiActions
{
	public function executeDelete()
	{
		$con = null;
		try{
			$pkg_id = mfwRequest::param('id');

			$app = $this->app;

			$pkg = PackageDb::retrieveByPK($pkg_id);
			if(!$pkg || $app->getId()!==$pkg->getAppId()){
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'Invalid package id'));
			}

			$con = mfwDBConnection::getPDO();
			$con->beginTransaction();

			$app = ApplicationDb::retrieveByPKForUpdate($app->getId(),$con);
			$pkg->delete($con);

			if($app->getLastUpload()==$pkg->getCreated()){
				// 最終アップデート時刻を前のものに戻す
				$pkg = PackageDb::selectNewestOneByAppId($app->getId());
				$lastupload = ($pkg)? $pkg->getCreated(): null;

				$app->updateLastUpload($lastupload,$con);
			}

			$con->commit();
		}
		catch(Exception $e){
			if($con) $con->rollback();
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}

		try{
			$pkg->deleteFile();
		}
		catch(Exception $e){
			// S3から削除出来なくてもDBからは消えているので許容する
		}

		apache_log('app_id',$app->getId());
		apache_log('pkg_id',$pkg->getId());

		return $this->jsonResponse(
			self::HTTP_200_OK,
			self::makePackageArray($pkg));
	}

}
