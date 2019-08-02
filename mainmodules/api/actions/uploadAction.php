<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/IPAFile.php';
require_once APP_ROOT.'/model/APKFile.php';
require_once APP_ROOT.'/model/AttachedFile.php';

class uploadAction extends apiActions
{
	public function executeUpload()
	{
		$con = null;
		$apkfile = null;
		try{
			if(mfwRequest::method()!=='POST'){
				return $this->jsonResponse(
					self::HTTP_405_METHODNOTALLOWED,
					array('error'=>'Method Not Allowed'));
			}

			$file_info = mfwRequest::param('file');
			$title = mfwRequest::param('title');
			$description = mfwRequest::param('description');
			$notify = self::parseBool(mfwRequest::param('notify'));
			$tag_names = explode(',',mfwRequest::param('tags'));
			$protect = self::parseBool(mfwRequest::param('protect'));
			$dsymfile = mfwRequest::param('dsym');
			if(!$file_info||!$title){
				$fields = array();
				if(!$file_info){
					$fields[] = 'file';
				}
				if(!$title){
					$fields[] = 'title';
				}
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'A required field ('.implode(',',$fields).') is not present.'));
			}
			if(!isset($file_info['error'])||$file_info['error']!==UPLOAD_ERR_OK){
				error_log(__METHOD__.'('.__LINE__.'): upload file error: $_FILES[file]='.json_encode($file_info));
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'upload file error: $_FILES[file]='.json_encode($file_info)));
			}

			$attached_files = array();

			// ファイルフォーマット確認, 情報抽出
			list($platform,$ext,$mime) = PackageDb::getPackageInfo(
				$file_info['name'],$file_info['tmp_name'],$file_info['type']);
			$identifier = null;
			if($platform===Package::PF_IOS){
				$plist = IPAFile::parseInfoPlist($file_info['tmp_name']);
				$identifier = $plist['CFBundleIdentifier'];
			}
			if($platform===Package::PF_ANDROID){
				if($ext==="aab"){
					$apkfile = APKFile::extractFromAppBundle($file_info['tmp_name']);
					$identifier = APKFile::getPackageName($apkfile);
					$attached_files[] = array(
						'filepath' => $apkfile,
						'original_name' => substr($file_info['name'], 0, -3).'apk',
						'type' => AttachedFile::TYPE_APK,
						'mime' => Package::MIME_ANDROID_APK,
						);
				}
				else{
					$identifier = APKFile::getPackageName($file_info['tmp_name']);
				}
			}

			if($dsymfile){
				$attached_files[] = array(
					'filepath' => $dsymfile['tmp_name'],
					'original_name' => $dsymfile['name'],
					'type' => AttachedFile::TYPE_DSYM,
					'mime' => $dsymfile['type'],
					);
			}

			// DBへ保存
			$con = mfwDBConnection::getPDO();
			$con->beginTransaction();

			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

			$tags = $app->getOrInsertTagsByName($tag_names,$con);

			$pkg = PackageDb::insertNewPackage(
				$app->getId(),$platform,$ext,
				$title,$description,$identifier,
				$file_info['name'],$file_info['size'],$tags,$protect,$con);
			apache_log('pkg_id',$pkg->getId());

			foreach($attached_files as $k => $afile){
				$attached_files[$k]['obj'] = AttachedFileDb::insertNewAttachedFile(
					$pkg,$afile['original_name'],filesize($afile['filepath']),$afile['type'],$con);
			}

			// S3へアップロード
			$pkg->uploadFile($file_info['tmp_name'],$mime);

			foreach($attached_files as $afile){
				$afile['obj']->uploadFile($afile['filepath'],$afile['mime']);
			}

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
		finally{
			if($apkfile){
				unlink($apkfile);
			}
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
			self::makePackageArray($pkg));
	}

}

