<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/IPAFile.php';

class upload_package_temporaryAction extends apiActions
{
	public function executeUpload_package_temporary()
	{
		try{
			$file_info = mfwRequest::param('file');
			if(!$file_info || !isset($file_info['error']) || $file_info['error']!=UPLOAD_ERR_OK){
				error_log(__METHOD__.'('.__LINE__.'): upload file error: $_FILES[file]='.json_encode($file_info));
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'upload_file error: $_FILES[file]='.json_encode($file_info)));
			}
			$file_name = $file_info['name'];
			$file_path = $file_info['tmp_name'];
			$file_type = $file_info['type'];

			list($platform,$ext,$mime) = PackageDb::getPackageInfo($file_name,$file_path,$file_type);

			$temp_name = Package::uploadTempFile($file_path,$ext,$mime);

			$ios_identifier = null;
			if($platform===Package::PF_IOS){
				$plist = IPAFile::parseInfoPlist($file_path);
				$ios_identifier = $plist['CFBundleIdentifier'];
			}
		}
		catch(Exception $e){
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}
		return $this->jsonResponse(
			self::HTTP_200_OK,
			array(
				'file_name' => $file_name,
				'temp_name' => $temp_name,
				'platform' => $platform,
				'ios_identifier' => $ios_identifier,
				));
	}
}
