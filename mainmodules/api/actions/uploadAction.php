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

			$app = ApplicationDb::selectByApiKey($api_key);
			if(!$app){
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'Invalid api_key'));
			}

			// ファイルフォーマット確認, 情報抽出
			$file_content = file_get_contents($file_info['tmp_name']);
			list($platform,$ext,$mime) = PackageDb::getPackageInfo(
				$file_info['name'],$file_content,$file_info['type']);
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
				$title,$description,$ios_identifier,$tags,$con);

			// S3へアップロード
			$pkg->uploadFile($file_content,$mime);

			$app->updateLastUpload($pkg->getCreated(),$con);

			$con->commit();
		}
		catch(Exception $e){
			if($con) $con->rollback();
			error_log(__METHOD__."$e->getMessage()");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage()));
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			array(
				'package_url' => mfwRequest::makeUrl("/package?id={$pkg->getId()}"),
				'application_url' => mfwRequest::makeUrl('/app?id={$app->getId()}'),
				'platform' => $platform,
				'created' => $pkg->getCreated(),
				));
	}

}

