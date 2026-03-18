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
		$temp_names = array(); // S3 tempファイル名 (エラー時の掃除用)
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
			$apkfile_param = mfwRequest::param('apk');
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
					if($apkfile_param){
						if(!isset($apkfile_param['error'])||$apkfile_param['error']!==UPLOAD_ERR_OK){
							error_log(__METHOD__.'('.__LINE__.'): upload apk file error: $_FILES[apk]='.json_encode($apkfile_param));
							return $this->jsonResponse(
								self::HTTP_400_BADREQUEST,
								array('error'=>'upload apk file error: $_FILES[apk]='.json_encode($apkfile_param)));
						}
						$identifier = APKFile::getPackageName($apkfile_param['tmp_name']);
						$attached_files[] = array(
							'filepath' => $apkfile_param['tmp_name'],
							'original_name' => $apkfile_param['name'],
							'type' => AttachedFile::TYPE_APK,
							'mime' => Package::MIME_ANDROID_APK,
							);
					}
					else{
						$apkfile = APKFile::extractFromAppBundle($file_info['tmp_name']);
						$identifier = APKFile::getPackageName($apkfile);
						$attached_files[] = array(
							'filepath' => $apkfile,
							'original_name' => substr($file_info['name'], 0, -3).'apk',
							'type' => AttachedFile::TYPE_APK,
							'mime' => Package::MIME_ANDROID_APK,
							);
					}
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

			// トランザクション外でtempにアップロード
			$pkg_temp_name = Package::uploadTempFile($file_info['tmp_name'],$ext,$mime);
			$temp_names[] = $pkg_temp_name;

			foreach($attached_files as $k => $afile){
				$afile_ext = pathinfo($afile['original_name'],PATHINFO_EXTENSION);
				$afile_temp = AttachedFile::uploadTempFile($afile['filepath'],$afile_ext,$afile['mime']);
				$attached_files[$k]['temp_name'] = $afile_temp;
				$temp_names[] = $afile_temp;
			}

			// DB トランザクション開始
			$con = $this->getPDO();
			$con->exec('SET innodb_lock_wait_timeout = 300');
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

			// tempから正しい場所へ移動
			$pkg->renameTempFile($pkg_temp_name);
			foreach($attached_files as $afile){
				$afile['obj']->renameTempFile($afile['temp_name']);
			}

			$temp_names = array(); // rename成功、掃除不要

			$app->updateLastUpload($pkg->getCreated(),$con);

			// DB トランザクション終了
			$con->commit();
		}
		catch(Exception $e){
			if($con) $con->rollback();
			// tempファイルの掃除
			foreach($temp_names as $tmp){
				try{ Storage::delete(Package::TEMP_DIR.$tmp); }catch(Exception $ignore){}
			}
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
