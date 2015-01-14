<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/IPAFile.php';

class uploadAction extends apiActions
{
	public function executeUpload()
	{
		$con = null;
		try{
			if(mfwRequest::method()!=='POST'){
				return $this->jsonResponse(
					self::HTTP_405_METHODNOTALLOWED,
					array('error'=>'Method Not Allowed'));
			}

			$api_key = mfwRequest::param('api_key');
			$file_info = mfwRequest::param('file');
			$title = mfwRequest::param('title');
			$description = mfwRequest::param('description');
			$notify = mfwRequest::param('notify');
			$tag_names = explode(',',mfwRequest::param('tags'));
			if(!$api_key||!$file_info||!$title){
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'A required field is not present.'));
			}
			if(!isset($file_info['error'])||$file_info['error']!==UPLOAD_ERR_OK){
				error_log(__METHOD__.'('.__LINE__.'): upload file error: $_FILES[file]='.json_encode($file_info));
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'upload file error: $_FILES[file]='.json_encode($file_info)));
			}

			$app = ApplicationDb::selectByApiKey($api_key);
			if(!$app){
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'Invalid api_key'));
			}
			apache_log('app_id',$app->getId());

			// ファイルフォーマット確認, 情報抽出
			list($platform,$ext,$mime) = PackageDb::getPackageInfo(
				$file_info['name'],$file_info['tmp_name'],$file_info['type']);
			$ios_identifier = null;
			if($platform===Package::PF_IOS){
				$plist = IPAFile::parseInfoPlist($file_info['tmp_name']);
				$ios_identifier = $plist['CFBundleIdentifier'];
			}

			// DBへ保存
			$con = mfwDBConnection::getPDO();
			$con->beginTransaction();

			$app = ApplicationDb::retrieveByPKForUpdate($app->getId());

			$tags = $app->getTagsByName($tag_names,$con);

			$pkg = PackageDb::insertNewPackage(
				$app->getId(),$platform,$ext,
				$title,$description,$ios_identifier,
				$file_info['name'],$file_info['size'],$tags,$con);
			apache_log('pkg_id',$pkg->getId());

			// S3へアップロード
			$pkg->uploadFile($file_info['tmp_name'],$mime);

			$app->updateLastUpload($pkg->getCreated(),$con);

			$con->commit();
		}
		catch(Exception $e){
			if($con) $con->rollback();
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}

		if($notify){
			try{
				$users = $app->getInstallUsers();
				$users->noticePackageUploaded($pkg);
			}
			catch(Exception $e){
				// アップロード通知が送れなくても許容する
				error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			}
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			$this->makePackageArray($pkg));
	}

}

